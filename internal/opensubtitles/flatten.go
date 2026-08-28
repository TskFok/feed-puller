package opensubtitles

import (
	"encoding/json"
	"slices"
	"strings"
)

func FlattenSearchData(raw []byte) []SubtitleFile {
	var payload struct {
		Data []struct {
			Attributes struct {
				Language       string  `json:"language"`
				DownloadCount  int     `json:"download_count"`
				Ratings        float64 `json:"ratings"`
				Release        string  `json:"release"`
				FeatureDetails struct {
					MovieName string `json:"movie_name"`
					Title     string `json:"title"`
				} `json:"feature_details"`
				Files []struct {
					FileID   int64  `json:"file_id"`
					FileName string `json:"file_name"`
				} `json:"files"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	var items []SubtitleFile
	for _, row := range payload.Data {
		release := strings.TrimSpace(row.Attributes.Release)
		if release == "" {
			release = strings.TrimSpace(row.Attributes.FeatureDetails.MovieName)
		}
		if release == "" {
			release = strings.TrimSpace(row.Attributes.FeatureDetails.Title)
		}
		for _, file := range row.Attributes.Files {
			if file.FileID <= 0 {
				continue
			}
			items = append(items, SubtitleFile{
				FileID:        file.FileID,
				FileName:      file.FileName,
				Release:       release,
				Language:      row.Attributes.Language,
				DownloadCount: row.Attributes.DownloadCount,
				Ratings:       row.Attributes.Ratings,
			})
		}
	}
	slices.SortStableFunc(items, func(a, b SubtitleFile) int {
		return b.DownloadCount - a.DownloadCount
	})
	return items
}
