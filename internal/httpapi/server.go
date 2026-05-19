package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/store"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	service *app.Service
	log     *slog.Logger
	handler http.Handler
}

func New(cfg config.Config, store *store.Store, service *app.Service, log *slog.Logger) *Server {
	server := &Server{cfg: cfg, store: store, service: service, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", server.handleLogin)
	mux.HandleFunc("/api/auth/logout", server.handleLogout)
	mux.HandleFunc("/api/auth/me", server.handleMe)
	mux.HandleFunc("/api/auth/feishu/login-url", server.handleFeishuLoginURL)
	mux.HandleFunc("/api/auth/feishu/login", server.handleFeishuLoginRedirect)
	mux.HandleFunc("/api/auth/feishu/start", server.handleFeishuStart)
	mux.HandleFunc("/api/auth/feishu/callback", server.handleFeishuCallback)
	mux.HandleFunc("/api/subscriptions/preview-next-poll", server.requireAuth(server.handleSubscriptionNextPollPreview))
	mux.HandleFunc("/api/subscriptions/reorder", server.requireAuth(server.handleSubscriptionReorder))
	mux.HandleFunc("/api/subscriptions", server.requireAuth(server.handleSubscriptions))
	mux.HandleFunc("/api/subscriptions/", server.requireAuth(server.handleSubscriptionByID))
	mux.HandleFunc("/api/items/batch-download", server.requireAuth(server.handleItemsBatchDownload))
	mux.HandleFunc("/api/items/batch-status", server.requireAuth(server.handleItemsBatchStatus))
	mux.HandleFunc("/api/items/", server.requireAuth(server.handleItemSubroutes))
	mux.HandleFunc("/api/items", server.requireAuth(server.handleItemsList))
	mux.HandleFunc("/api/downloads/active", server.requireAuth(server.handleActiveDownloads))
	mux.HandleFunc("/api/downloads/completed", server.requireAuth(server.handleCompletedDownloads))
	mux.HandleFunc("/api/downloads", server.requireAuth(server.handleDownloads))
	mux.HandleFunc("/api/downloads/", server.requireAuth(server.handleDownloadByID))
	mux.HandleFunc("/api/ai-configs", server.requireAuth(server.handleAIConfigs))
	mux.HandleFunc("/api/ai-configs/", server.requireAuth(server.handleAIConfigByID))
	mux.HandleFunc("/api/settings/proxy", server.requireAuth(server.handleProxySetting))
	mux.HandleFunc("/api/settings/feishu-binding", server.requireAuth(server.handleFeishuBinding))
	mux.HandleFunc("/api/settings/feishu-bind-url", server.requireAuth(server.handleFeishuBindURL))
	mux.HandleFunc("/", server.handleStatic)
	server.handler = server.withOptionalUser(mux)
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}
	staticDir := s.cfg.StaticDir
	if _, err := os.Stat(staticDir); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("feed-puller API 正在运行。请先构建前端或设置 STATIC_DIR。"))
		return
	}
	requestPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if requestPath == "." {
		requestPath = "index.html"
	}
	fullPath := filepath.Join(staticDir, requestPath)
	if !strings.HasPrefix(fullPath, filepath.Clean(staticDir)) {
		writeError(w, http.StatusBadRequest, "路径无效")
		return
	}
	if stat, err := os.Stat(fullPath); err != nil || stat.IsDir() {
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		return
	}
	http.ServeFile(w, r, fullPath)
}
