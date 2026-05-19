package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func (s *Server) handleFeishuLoginURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.cfg.FeishuAppID == "" || s.cfg.FeishuAppSecret == "" {
		writeError(w, http.StatusBadRequest, "飞书应用未配置")
		return
	}
	state := "login"
	writeJSON(w, http.StatusOK, map[string]string{
		"url":  "/api/auth/feishu/login?state=" + state,
		"goto": feishuPassportAuthorizeURLFor(s.cfg.BaseURL, s.cfg.FeishuAppID, state),
	})
}

func (s *Server) handleFeishuLoginRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.cfg.FeishuAppID == "" || s.cfg.FeishuAppSecret == "" {
		writeError(w, http.StatusBadRequest, "飞书应用未配置")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		state = "login"
	}
	http.Redirect(w, r, feishuPassportAuthorizeURLFor(s.cfg.BaseURL, s.cfg.FeishuAppID, state), http.StatusFound)
}

func (s *Server) handleFeishuStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	state := "login"
	if user, ok := s.currentUser(r); ok {
		state = fmt.Sprintf("bind:%d", user.ID)
	}
	http.Redirect(w, r, "/api/auth/feishu/login?state="+url.QueryEscape(state), http.StatusFound)
}

func (s *Server) handleFeishuBindURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	user, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	if s.cfg.FeishuAppID == "" || s.cfg.FeishuAppSecret == "" {
		writeError(w, http.StatusBadRequest, "飞书应用未配置")
		return
	}
	state := fmt.Sprintf("bind:%d", user.ID)
	out := map[string]string{
		"url": "/api/auth/feishu/login?state=" + state,
	}
	if s.cfg.FeishuAppID != "" {
		out["goto"] = feishuPassportAuthorizeURLFor(s.cfg.BaseURL, s.cfg.FeishuAppID, state)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleFeishuCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" {
		writeError(w, http.StatusBadRequest, "缺少 code")
		return
	}
	identity, err := s.exchangeFeishuCode(r.Context(), code)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.HasPrefix(state, "bind:") {
			_, _ = w.Write([]byte(feishuBindCallbackHTML("feishu_bind_error", jsonString(err.Error()))))
			return
		}
		_, _ = w.Write([]byte(feishuLoginCallbackHTML("feishu_login_error", "", jsonString(err.Error()))))
		return
	}

	if strings.HasPrefix(state, "bind:") {
		var userID int64
		if _, err := fmt.Sscanf(state, "bind:%d", &userID); err != nil || userID <= 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(feishuBindCallbackHTML("feishu_bind_error", jsonString("无效的绑定 state"))))
			return
		}
		if existing, err := s.store.UserByFeishuOpenID(r.Context(), identity.OpenID); err == nil && existing.ID != userID {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(feishuBindCallbackHTML("feishu_bind_error", jsonString("该飞书账号已绑定其他用户"))))
			return
		}
		if err := s.store.BindFeishu(r.Context(), userID, identity.OpenID, identity.Name); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(feishuBindCallbackHTML("feishu_bind_error", jsonString(err.Error()))))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(feishuBindCallbackHTML("feishu_bind_success", "")))
		return
	}

	user, err := s.store.UserByFeishuOpenID(r.Context(), identity.OpenID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(feishuLoginCallbackHTML("feishu_login_error", "", jsonString("该飞书账号尚未绑定管理员"))))
		return
	}
	if err := s.setSession(r.Context(), w, user.ID); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(feishuLoginCallbackHTML("feishu_login_error", "", jsonString(err.Error()))))
		return
	}
	userJSON, _ := json.Marshal(sanitizeUser(user))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(feishuLoginCallbackHTML("feishu_login_success", string(userJSON), "")))
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
