// Package fetch 抓取并解析猫眼在映票房榜。
package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DashboardURL 猫眼专业版在映票房接口。
const DashboardURL = "https://piaofang.maoyan.com/dashboard-ajax/movie"

// Film 一部在映片（快照条目）。
type Film struct {
	MovieID       int64   `json:"movie_id"`
	Name          string  `json:"name"`
	ReleaseInfo   string  `json:"release_info"`
	SumBoxDesc    string  `json:"sum_box_desc"`
	SumBoxWan     float64 `json:"sum_box_wan"` // 万元
	BoxRate       string  `json:"box_rate"`
	ShowCount     int     `json:"show_count"`
	ShowCountRate string  `json:"show_count_rate"`
	AvgShowView   string  `json:"avg_show_view"`
	AvgSeatView   string  `json:"avg_seat_view"`
}

var sumBoxRe = regexp.MustCompile(`^([0-9.]+)(亿|万)?$`)

// ParseSumBox 将猫眼累计票房文案解析为万元。
func ParseSumBox(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0, fmt.Errorf("空累计票房")
	}
	m := sumBoxRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("无法解析累计票房: %q", s)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	switch m[2] {
	case "亿":
		return v * 10000, nil // 亿 → 万元
	case "万":
		return v, nil
	default:
		return v / 10000, nil // 元 → 万元
	}
}

// ParseDashboard 解析猫眼 dashboard-ajax/movie 响应。
func ParseDashboard(raw []byte) ([]Film, error) {
	var root struct {
		MovieList struct {
			List []struct {
				AvgSeatView   string `json:"avgSeatView"`
				AvgShowView   string `json:"avgShowView"`
				BoxRate       string `json:"boxRate"`
				ShowCount     int    `json:"showCount"`
				ShowCountRate string `json:"showCountRate"`
				SumBoxDesc    string `json:"sumBoxDesc"`
				MovieInfo     struct {
					MovieID     int64  `json:"movieId"`
					MovieName   string `json:"movieName"`
					ReleaseInfo string `json:"releaseInfo"`
				} `json:"movieInfo"`
			} `json:"list"`
		} `json:"movieList"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("解析猫眼 JSON: %w", err)
	}
	out := make([]Film, 0, len(root.MovieList.List))
	for _, it := range root.MovieList.List {
		wan, err := ParseSumBox(it.SumBoxDesc)
		if err != nil {
			return nil, fmt.Errorf("影片 %s: %w", it.MovieInfo.MovieName, err)
		}
		out = append(out, Film{
			MovieID:       it.MovieInfo.MovieID,
			Name:          it.MovieInfo.MovieName,
			ReleaseInfo:   it.MovieInfo.ReleaseInfo,
			SumBoxDesc:    it.SumBoxDesc,
			SumBoxWan:     wan,
			BoxRate:       it.BoxRate,
			ShowCount:     it.ShowCount,
			ShowCountRate: it.ShowCountRate,
			AvgShowView:   it.AvgShowView,
			AvgSeatView:   it.AvgSeatView,
		})
	}
	return out, nil
}

// Fetch 请求猫眼；失败重试共 3 次（1 次初始 + 2 次重试）。
func Fetch(ctx context.Context, client *http.Client) ([]Film, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		films, err := fetchOnce(ctx, client)
		if err == nil {
			return films, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}
	return nil, last
}

func fetchOnce(ctx context.Context, client *http.Client) ([]Film, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DashboardURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://piaofang.maoyan.com/dashboard")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("猫眼 HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return ParseDashboard(body)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
