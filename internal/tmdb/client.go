package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.themoviedb.org/3"

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type MovieDetails struct {
	Title string
	Year  int
}

type TVDetails struct {
	Name string
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c.apiKey != ""
}

func (c *Client) GetMovieDetails(ctx context.Context, tmdbID, imdbID int64) (MovieDetails, error) {
	if !c.Enabled() {
		return MovieDetails{}, fmt.Errorf("TMDB API Key 未配置")
	}
	if tmdbID > 0 {
		return c.getMovieByID(ctx, tmdbID)
	}
	imdb := FormatIMDbID(imdbID)
	if imdb == "" {
		return MovieDetails{}, fmt.Errorf("缺少 TMDB/IMDb ID")
	}
	return c.findMovieByIMDb(ctx, imdb)
}

func (c *Client) GetTVDetails(ctx context.Context, tmdbID, tvdbID int64) (TVDetails, error) {
	if !c.Enabled() {
		return TVDetails{}, fmt.Errorf("TMDB API Key 未配置")
	}
	if tmdbID > 0 {
		return c.getTVByID(ctx, tmdbID)
	}
	if tvdbID <= 0 {
		return TVDetails{}, fmt.Errorf("缺少 TMDB/TVDB ID")
	}
	return c.findTVByTVDB(ctx, tvdbID)
}

func (c *Client) getMovieByID(ctx context.Context, tmdbID int64) (MovieDetails, error) {
	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s&language=zh-CN", c.baseURL, tmdbID, url.QueryEscape(c.apiKey))
	var payload struct {
		Title         string `json:"title"`
		OriginalTitle string `json:"original_title"`
		ReleaseDate   string `json:"release_date"`
	}
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return MovieDetails{}, err
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = strings.TrimSpace(payload.OriginalTitle)
	}
	return MovieDetails{Title: title, Year: parseYear(payload.ReleaseDate)}, nil
}

func (c *Client) findMovieByIMDb(ctx context.Context, imdb string) (MovieDetails, error) {
	endpoint := fmt.Sprintf("%s/find/%s?api_key=%s&external_source=imdb_id&language=zh-CN", c.baseURL, url.PathEscape(imdb), url.QueryEscape(c.apiKey))
	var payload struct {
		MovieResults []struct {
			ID          int64  `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
		} `json:"movie_results"`
	}
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return MovieDetails{}, err
	}
	if len(payload.MovieResults) == 0 {
		return MovieDetails{}, fmt.Errorf("TMDB 未找到电影 %s", imdb)
	}
	first := payload.MovieResults[0]
	if first.ID > 0 {
		return c.getMovieByID(ctx, first.ID)
	}
	return MovieDetails{Title: first.Title, Year: parseYear(first.ReleaseDate)}, nil
}

func (c *Client) getTVByID(ctx context.Context, tmdbID int64) (TVDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d?api_key=%s&language=zh-CN", c.baseURL, tmdbID, url.QueryEscape(c.apiKey))
	var payload struct {
		Name         string `json:"name"`
		OriginalName string `json:"original_name"`
	}
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return TVDetails{}, err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = strings.TrimSpace(payload.OriginalName)
	}
	if name == "" {
		return TVDetails{}, fmt.Errorf("TMDB 未返回剧集名称")
	}
	return TVDetails{Name: name}, nil
}

func (c *Client) findTVByTVDB(ctx context.Context, tvdbID int64) (TVDetails, error) {
	endpoint := fmt.Sprintf("%s/find/%d?api_key=%s&external_source=tvdb_id&language=zh-CN", c.baseURL, tvdbID, url.QueryEscape(c.apiKey))
	var payload struct {
		TVResults []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"tv_results"`
	}
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return TVDetails{}, err
	}
	if len(payload.TVResults) == 0 {
		return TVDetails{}, fmt.Errorf("TMDB 未找到 TVDB %d", tvdbID)
	}
	if payload.TVResults[0].ID > 0 {
		return c.getTVByID(ctx, payload.TVResults[0].ID)
	}
	name := strings.TrimSpace(payload.TVResults[0].Name)
	if name == "" {
		return TVDetails{}, fmt.Errorf("TMDB 未返回剧集名称")
	}
	return TVDetails{Name: name}, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("创建 TMDB 请求失败: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 TMDB 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("TMDB 返回 HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("TMDB 返回 HTTP %d: %s", resp.StatusCode, msg)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("解析 TMDB 响应失败: %w", err)
	}
	return nil
}

func FormatIMDbID(id int64) string {
	if id <= 0 {
		return ""
	}
	return "tt" + strconv.FormatInt(id, 10)
}

func parseYear(releaseDate string) int {
	releaseDate = strings.TrimSpace(releaseDate)
	if len(releaseDate) < 4 {
		return 0
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil {
		return 0
	}
	return year
}
