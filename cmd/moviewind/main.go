// moviewind：A股/港股影视公司观察工具。
//
// 用法：
//
//	moviewind run                 抓取 → 分析 → 报告 → 推送
//	moviewind run --dry-run       同上但不推送
//	moviewind run --date 2026-08-21  用已有快照重跑分析（不重新抓取）
//	moviewind fetch               只抓取并入库
//	moviewind report [--date]     用已有快照生成报告（可推送）
//	moviewind list                列出已有快照
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"movie-wind/internal/analyze"
	"movie-wind/internal/config"
	"movie-wind/internal/fetch"
	"movie-wind/internal/mapping"
	"movie-wind/internal/notify"
	"movie-wind/internal/report"
	"movie-wind/internal/store"
)

const (
	configPath     = "config.yaml"
	localCfgPath   = "config.local.yaml"
	moviesFile     = "data/movies.yaml"
	defaultDataDir = "data"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func usage() {
	fmt.Fprintln(os.Stderr, `moviewind — A股/港股影视公司观察工具

用法:
  moviewind run                 抓取 → 分析 → 报告 → 推送
  moviewind fetch               只抓取入库
  moviewind report              用已有快照生成报告（可推送）
  moviewind list                列出已有快照

选项:
  --dry-run                    不推送企业微信
  --date YYYY-MM-DD            指定日期（run 带 date 时不抓取，只用已有快照）
  --data-dir DIR               数据目录（默认 data/）
  --movies FILE                映射表路径（默认 data/movies.yaml）`)
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	cmd, opts := args[0], parseOpts(args[1:])
	switch cmd {
	case "run":
		return cmdRun(opts)
	case "fetch":
		return cmdFetch(opts)
	case "report":
		return cmdReport(opts)
	case "list":
		return cmdList(opts)
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		usage()
		return 2
	}
}

func parseOpts(args []string) map[string]string {
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			continue
		}
		key := strings.TrimPrefix(a, "--")
		if eq := strings.Index(key, "="); eq >= 0 {
			opts[key[:eq]] = key[eq+1:]
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			opts[key] = args[i+1]
			i++
		} else {
			opts[key] = ""
		}
	}
	return opts
}

func cmdRun(opts map[string]string) int {
	cfg, tab, err := loadBase(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	date := opts["date"]
	dir := dataDir(opts)
	if date == "" {
		date = time.Now().Format("2006-01-02")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		snap, err := doFetch(ctx, date)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := snap.Save(dir); err != nil {
			fmt.Fprintln(os.Stderr, "保存快照失败:", err)
			return 1
		}
		return analyzeReportAndNotify(cfg, tab, snap, date, opts)
	}
	snap, err := store.Load(dir, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取快照失败:", err)
		return 1
	}
	return analyzeReportAndNotify(cfg, tab, snap, date, opts)
}

func cmdFetch(opts map[string]string) int {
	date := opts["date"]
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	snap, err := doFetch(ctx, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := snap.Save(dataDir(opts)); err != nil {
		fmt.Fprintln(os.Stderr, "保存快照失败:", err)
		return 1
	}
	fmt.Printf("已保存快照 %s：%d 部影片", date, len(snap.Films))
	if snap.FetchError != "" {
		fmt.Printf("（抓取错误：%s）", snap.FetchError)
	}
	fmt.Println()
	return 0
}

func cmdReport(opts map[string]string) int {
	cfg, tab, err := loadBase(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dir := dataDir(opts)
	date := opts["date"]
	if date == "" {
		dates, err := store.ListDates(dir)
		if err != nil || len(dates) == 0 {
			fmt.Fprintln(os.Stderr, "没有可用快照，请先运行 moviewind run / fetch")
			return 1
		}
		date = dates[len(dates)-1]
	}
	snap, err := store.Load(dir, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取快照失败:", err)
		return 1
	}
	return analyzeReportAndNotify(cfg, tab, snap, date, opts)
}

func cmdList(opts map[string]string) int {
	dates, err := store.ListDates(dataDir(opts))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, d := range dates {
		fmt.Println(d)
	}
	return 0
}

func loadBase(opts map[string]string) (*config.Config, *mapping.Table, error) {
	cfg, err := config.Load(configPath, localCfgPath)
	if err != nil {
		return nil, nil, err
	}
	movies := opts["movies"]
	if movies == "" {
		movies = moviesFile
	}
	tab, err := mapping.Load(movies)
	if err != nil {
		return nil, nil, err
	}
	return cfg, tab, nil
}

func dataDir(opts map[string]string) string {
	if d := opts["data-dir"]; d != "" {
		return d
	}
	return defaultDataDir
}

func doFetch(ctx context.Context, date string) (*store.Snapshot, error) {
	snap := &store.Snapshot{Date: date, FetchedAt: time.Now()}
	films, err := fetch.Fetch(ctx, nil)
	if err != nil {
		snap.FetchError = err.Error()
		return snap, fmt.Errorf("抓取失败: %w", err)
	}
	snap.Films = films
	return snap, nil
}

func analyzeReportAndNotify(cfg *config.Config, tab *mapping.Table, snap *store.Snapshot, date string, opts map[string]string) int {
	dir := dataDir(opts)
	prev, err := store.LatestBefore(dir, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var prevPrev *store.Snapshot
	if prev != nil {
		prevPrev, err = store.LatestBefore(dir, prev.Date)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	res := analyze.Analyze(snap, prev, prevPrev, tab, analyze.Options{
		TopN:            cfg.TopN,
		SummaryRankMove: cfg.SummaryRankMove,
	})
	md := report.Build(date, res, tab, snap.FetchError, res.Baseline)

	reportDir := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reportDir, 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(reportDir, date+".md"), []byte(md), 0o644)
	}
	fmt.Println(md)

	if _, dry := opts["dry-run"]; !dry && cfg.Notify.WebhookURL != "" {
		// notify_when_quiet 默认 true：即使无总结变化也推完整报告
		if !cfg.Notify.NotifyWhenQuiet && !res.Baseline && len(res.Summary) == 0 {
			fmt.Println("ℹ️ 今日无总结变化，不推送（notify_when_quiet=false）")
		} else if err := notifyAll(cfg, date, res, md, dir); err != nil {
			fmt.Fprintln(os.Stderr, "推送失败:", err)
			return 1
		}
	}
	return 0
}

func notifyAll(cfg *config.Config, date string, res *analyze.Result, md, dir string) error {
	w := notify.New(cfg.Notify.WebhookURL)
	if w == nil {
		return nil
	}
	footer := fmt.Sprintf("\n---\n完整报告：data/reports/%s.md", date)
	kind := "report"
	var msgs []string
	switch {
	case res.Baseline:
		kind = "baseline"
		msgs = []string{fmt.Sprintf("# 电影公司观察 · %s\n\n基线已建立：今日在映快照已入库，下次运行将计算日票房 Top10。%s", date, footer)}
	default:
		var err error
		msgs, err = notify.BuildMessages("", md, footer)
		if err != nil {
			return err
		}
	}
	sum := sha256.Sum256([]byte(md + "\n" + kind))
	hash := hex.EncodeToString(sum[:8])
	st := store.LoadState(dir)
	if st.LastSnapshotDate == date && st.LastNotifiedHash == hash {
		fmt.Println("ℹ️ 今天已推送过相同内容，跳过")
		return nil
	}
	if err := w.SendAll(msgs); err != nil {
		return err
	}
	return store.SaveState(dir, store.State{LastSnapshotDate: date, LastNotifiedHash: hash})
}
