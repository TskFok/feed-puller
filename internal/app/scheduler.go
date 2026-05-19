package app

import (
	"context"
	"log/slog"
	"time"

	"feed-puller/internal/store"
)

type Scheduler struct {
	store   *store.Store
	service *Service
	log     *slog.Logger
}

func NewScheduler(store *store.Store, service *Service, log *slog.Logger) *Scheduler {
	return &Scheduler{store: store, service: service, log: log}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	subscriptions, err := s.store.DueSubscriptions(ctx, time.Now().UTC())
	if err != nil {
		s.log.Error("查询待拉取订阅失败", "error", err)
		return
	}
	for _, sub := range subscriptions {
		if _, err := s.service.PollSubscription(ctx, sub); err != nil {
			s.log.Warn("订阅拉取失败", "subscription_id", sub.ID, "error", err)
		}
	}
	if err := s.service.SubmitPendingDownloads(ctx); err != nil {
		s.log.Warn("提交下载队列失败", "error", err)
	}
}
