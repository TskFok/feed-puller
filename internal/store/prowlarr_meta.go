package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ProwlarrMediaMovie = "movie"
	ProwlarrMediaTV    = "tv"
)

// ProwlarrItemMeta 存储在 feed_items.link 中的 Prowlarr 元数据。
type ProwlarrItemMeta struct {
	MediaType string `json:"media_type"`
	ImdbID    int64  `json:"imdb_id,omitempty"`
	TmdbID    int64  `json:"tmdb_id,omitempty"`
	TvdbID    int64  `json:"tvdb_id,omitempty"`
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
}

func EncodeProwlarrItemMeta(meta ProwlarrItemMeta) string {
	meta.MediaType = strings.TrimSpace(meta.MediaType)
	if meta.MediaType == "" {
		meta.MediaType = ProwlarrMediaMovie
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return ""
	}
	return string(raw)
}

func ParseProwlarrItemMeta(raw string) (ProwlarrItemMeta, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return ProwlarrItemMeta{}, false
	}
	var meta ProwlarrItemMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return ProwlarrItemMeta{}, false
	}
	if strings.TrimSpace(meta.MediaType) == "" {
		meta.MediaType = ProwlarrMediaMovie
	}
	return meta, true
}

func ParseProwlarrIndexerIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("索引器 ID 格式无效")
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			out = append(out, id)
		}
	}
	return out, nil
}

func EncodeProwlarrIndexerIDs(ids []int64) string {
	if len(ids) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
