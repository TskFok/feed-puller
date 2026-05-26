package prowlarr

import (
	"sort"
	"strings"
	"time"
)

type SortBy string

const (
	SortBySeeders SortBy = "seeders"
	SortBySize    SortBy = "size"
	SortByDate    SortBy = "date"
)

func NormalizeSortBy(raw string) SortBy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(SortBySize):
		return SortBySize
	case string(SortByDate):
		return SortByDate
	default:
		return SortBySeeders
	}
}

func SortReleases(releases []Release, sortBy SortBy) {
	switch sortBy {
	case SortBySize:
		sort.Slice(releases, func(i, j int) bool {
			return releases[i].Size > releases[j].Size
		})
	case SortByDate:
		sort.Slice(releases, func(i, j int) bool {
			return releaseTime(releases[i]).After(releaseTime(releases[j]))
		})
	default:
		sort.Slice(releases, func(i, j int) bool {
			if releases[i].Seeders == releases[j].Seeders {
				return releases[i].Size > releases[j].Size
			}
			return releases[i].Seeders > releases[j].Seeders
		})
	}
}

func releaseTime(release Release) time.Time {
	if release.PublishDate.IsZero() {
		return time.Time{}
	}
	return release.PublishDate
}
