package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"feed-puller/internal/store"
)

const sessionCookieName = "feed_puller_session"

type contextKey string

const userContextKey contextKey = "user"

func randomToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Server) setSession(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.store.CreateSession(ctx, token, userID, expiresAt); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.BaseURL, "https://"),
	})
	return nil
}

func (s *Server) clearSession(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.store.DeleteSession(ctx, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.BaseURL, "https://"),
	})
}

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	user, ok := r.Context().Value(userContextKey).(store.User)
	return user, ok
}

func (s *Server) withOptionalUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, err := s.store.UserBySession(r.Context(), cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.currentUser(r); !ok {
			writeError(w, http.StatusUnauthorized, "未登录")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	user, err := s.store.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}
	if err := s.setSession(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sanitizeUser(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	s.clearSession(r.Context(), w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	user, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	writeJSON(w, http.StatusOK, sanitizeUser(user))
}

func sanitizeUser(user store.User) map[string]any {
	return map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"feishu_bound":   user.FeishuOpenID != "",
		"feishu_name":    user.FeishuName,
		"feishu_open_id": user.FeishuOpenID,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	}
}

func (s *Server) handleFeishuStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.cfg.FeishuAppID == "" || s.cfg.FeishuAppSecret == "" {
		writeError(w, http.StatusBadRequest, "飞书应用未配置")
		return
	}
	mode := "login"
	userID := int64(0)
	if user, ok := s.currentUser(r); ok {
		mode = "bind"
		userID = user.ID
	}
	state := s.signOAuthState(mode, userID, time.Now())
	redirectURI := s.cfg.BaseURL + "/api/auth/feishu/callback"
	values := url.Values{}
	values.Set("app_id", s.cfg.FeishuAppID)
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)
	http.Redirect(w, r, "https://open.feishu.cn/open-apis/authen/v1/authorize?"+values.Encode(), http.StatusFound)
}

func (s *Server) handleFeishuCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	mode, stateUserID, err := s.verifyOAuthState(state)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	identity, err := s.exchangeFeishuCode(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if mode == "bind" {
		user, ok := s.currentUser(r)
		if !ok || user.ID != stateUserID {
			writeError(w, http.StatusUnauthorized, "绑定飞书需要先登录")
			return
		}
		if err := s.store.BindFeishu(r.Context(), user.ID, identity.OpenID, identity.Name); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		http.Redirect(w, r, "/settings?feishu=bound", http.StatusFound)
		return
	}
	user, err := s.store.UserByFeishuOpenID(r.Context(), identity.OpenID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "该飞书账号尚未绑定管理员")
		return
	}
	if err := s.setSession(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleFeishuBinding(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"bound":          user.FeishuOpenID != "",
			"feishu_open_id": user.FeishuOpenID,
			"feishu_name":    user.FeishuName,
		})
	case http.MethodDelete:
		if err := s.store.UnbindFeishu(r.Context(), user.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) signOAuthState(mode string, userID int64, now time.Time) string {
	payload := fmt.Sprintf("%s|%d|%d", mode, userID, now.Unix())
	mac := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	_, _ = mac.Write([]byte(payload))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (s *Server) verifyOAuthState(state string) (string, int64, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("OAuth state 无效")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", 0, fmt.Errorf("OAuth state 无效")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("OAuth state 无效")
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	_, _ = mac.Write(payloadBytes)
	if !hmac.Equal(mac.Sum(nil), signature) {
		return "", 0, fmt.Errorf("OAuth state 签名无效")
	}
	fields := strings.Split(string(payloadBytes), "|")
	if len(fields) != 3 {
		return "", 0, fmt.Errorf("OAuth state 无效")
	}
	createdAt, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || time.Since(time.Unix(createdAt, 0)) > 10*time.Minute {
		return "", 0, fmt.Errorf("OAuth state 已过期")
	}
	userID, _ := strconv.ParseInt(fields[1], 10, 64)
	return fields[0], userID, nil
}
