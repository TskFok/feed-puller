package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/downloader"
	"feed-puller/internal/httpapi"
	"feed-puller/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Error("打开 MySQL 失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.PingContext(ctx); err != nil {
		log.Error("连接 MySQL 失败", "error", err)
		os.Exit(1)
	}

	repo := store.New(db)
	if err := repo.Migrate(ctx); err != nil {
		log.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
	if err := repo.BootstrapAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		log.Error("初始化管理员失败", "error", err)
		os.Exit(1)
	}

	aria2 := downloader.NewAria2Client(cfg.Aria2RPCURL, cfg.Aria2RPCSecret)
	service := app.NewService(repo, aria2, log)
	scheduler := app.NewScheduler(repo, service, log)
	go scheduler.Run(ctx)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.New(cfg, repo, service, log),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Info("feed-puller 已启动", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("HTTP 服务异常退出", "error", err)
		os.Exit(1)
	}
}
