package prowlarr

import "time"

// Release 表示 Prowlarr 搜索返回的一条 release。
type Release struct {
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Indexer     string    `json:"indexer"`
	IndexerID   int64     `json:"indexerId"`
	Size        int64     `json:"size"`
	Seeders     int       `json:"seeders"`
	Leechers    int       `json:"leechers"`
	PublishDate time.Time `json:"publishDate"`
	DownloadURL string    `json:"downloadUrl"`
	InfoURL     string    `json:"infoUrl"`
	InfoHash    string    `json:"infoHash"`
	Protocol    string    `json:"protocol"`
	ImdbID      int64     `json:"imdbId"`
	TmdbID      int64     `json:"tmdbId"`
	TvdbID      int64     `json:"tvdbId"`
	Season      int       `json:"season"`
	Episode     int       `json:"episode"`
}
