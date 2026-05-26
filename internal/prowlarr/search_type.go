package prowlarr

import "strings"

type SearchType string

const (
	SearchTypeMovie SearchType = "movie"
	SearchTypeTV    SearchType = "tv"
)

func NormalizeSearchType(raw string) SearchType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(SearchTypeTV), "tvsearch", "series":
		return SearchTypeTV
	default:
		return SearchTypeMovie
	}
}

func (t SearchType) APIType() string {
	if t == SearchTypeTV {
		return "tvsearch"
	}
	return "moviesearch"
}

func (t SearchType) Category() string {
	if t == SearchTypeTV {
		return "5000"
	}
	return "2000"
}
