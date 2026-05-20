package rename

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ScrapeInput 刮削重命名所需的输入。
type ScrapeInput struct {
	FilePath           string
	Filename           string
	Title              string
	SubscriptionSeason int
	EpisodeOffset      int
	AI                 *AnimeExtract
	LocalEpisode       int
	LocalEpisodeOK     bool
}

// AnimeExtract 与 AI 识别结果对应的字段子集（季数由订阅配置决定，不在此携带）。
type AnimeExtract struct {
	AnimeName string
	Episode   int
}

// ScrapeTarget 解析后的刮削目标。
type ScrapeTarget struct {
	AnimeName string
	Season    int
	Episode   int
	Path      string
}

// ResolveScrapeTarget 解析番剧名、季数、集数并生成目标路径。
// 季数仅使用订阅配置（SubscriptionSeason），不使用目录或 AI 返回的季数；集数在识别结果上应用偏移。
func ResolveScrapeTarget(in ScrapeInput) (ScrapeTarget, error) {
	from := strings.TrimSpace(in.FilePath)
	if from == "" {
		return ScrapeTarget{}, fmt.Errorf("文件路径不能为空")
	}

	episode, err := resolveDetectedEpisode(in)
	if err != nil {
		return ScrapeTarget{}, err
	}
	finalEpisode, err := FinalEpisode(episode, in.EpisodeOffset)
	if err != nil {
		return ScrapeTarget{}, err
	}

	season := in.SubscriptionSeason
	if season < 1 {
		season = 1
	}

	animeName := resolveAnimeName(in)
	if animeName == "" {
		return ScrapeTarget{}, fmt.Errorf("无法确定番剧名")
	}

	targetPath := BuildScrapeTargetPath(from, animeName, season, finalEpisode)
	return ScrapeTarget{
		AnimeName: animeName,
		Season:    season,
		Episode:   finalEpisode,
		Path:      targetPath,
	}, nil
}

func resolveDetectedEpisode(in ScrapeInput) (int, error) {
	if in.AI != nil && in.AI.Episode > 0 {
		return in.AI.Episode, nil
	}
	if in.LocalEpisodeOK && in.LocalEpisode > 0 {
		return in.LocalEpisode, nil
	}
	return 0, fmt.Errorf("未能识别集数")
}

func resolveAnimeName(in ScrapeInput) string {
	if in.AI != nil {
		if name := SanitizeFilenamePart(in.AI.AnimeName); name != "" {
			return name
		}
	}
	base := strings.TrimSpace(in.Filename)
	if base == "" {
		base = strings.TrimSpace(in.Title)
	}
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	cleaned := StripEpisodeSuffix(base)
	if cleaned == "" {
		cleaned = base
	}
	return SanitizeFilenamePart(cleaned)
}
