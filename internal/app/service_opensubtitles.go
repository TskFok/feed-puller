package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"feed-puller/internal/opensubtitles"
	"feed-puller/internal/store"
)

var (
	ErrOpenSubtitlesNotConfigured = errors.New("OpenSubtitles 未配置")
	ErrSubtitleWriteFailed        = errors.New("保存字幕失败")
)

func (s *Service) GetOpenSubtitlesConfig(ctx context.Context) (store.OpenSubtitlesConfig, error) {
	return s.store.GetOpenSubtitlesConfig(ctx)
}

func (s *Service) SaveOpenSubtitlesConfig(ctx context.Context, cfg store.OpenSubtitlesConfig) (store.OpenSubtitlesConfig, error) {
	return s.store.SaveOpenSubtitlesConfig(ctx, cfg)
}

func (s *Service) SearchSubtitles(ctx context.Context, query, language string) ([]opensubtitles.SubtitleFile, error) {
	cfg, err := s.store.GetOpenSubtitlesConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Configured {
		return nil, ErrOpenSubtitlesNotConfigured
	}
	return s.opensubtitlesClientFor(cfg).Search(ctx, query, language)
}

func (s *Service) DownloadSubtitle(ctx context.Context, fileID int64, fallbackFileName string) (string, string, error) {
	cfg, err := s.store.GetOpenSubtitlesConfig(ctx)
	if err != nil {
		return "", "", err
	}
	if !cfg.Configured {
		return "", "", ErrOpenSubtitlesNotConfigured
	}
	client := s.opensubtitlesClientFor(cfg)
	info, err := client.RequestDownload(ctx, fileID)
	if err != nil {
		return "", "", err
	}
	name := strings.TrimSpace(info.FileName)
	if name == "" {
		name = fallbackFileName
	}
	sanitized, err := opensubtitles.SanitizeFileName(name)
	if err != nil {
		return "", "", err
	}
	dest := filepath.Join(cfg.DownloadDir, sanitized)
	body, err := client.FetchFile(ctx, info.Link)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(dest, body, 0o644); err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrSubtitleWriteFailed, err)
	}
	return dest, sanitized, nil
}

func (s *Service) opensubtitlesClientFor(cfg store.OpenSubtitlesConfig) *opensubtitles.Client {
	s.opensubtitlesMu.Lock()
	defer s.opensubtitlesMu.Unlock()
	if s.opensubtitlesClient != nil &&
		s.opensubtitlesUser == cfg.Username &&
		s.opensubtitlesPass == cfg.Password &&
		s.opensubtitlesKey == cfg.APIKey {
		return s.opensubtitlesClient
	}
	s.opensubtitlesClient = opensubtitles.NewClient(cfg.Username, cfg.Password, cfg.APIKey)
	s.opensubtitlesUser = cfg.Username
	s.opensubtitlesPass = cfg.Password
	s.opensubtitlesKey = cfg.APIKey
	return s.opensubtitlesClient
}
