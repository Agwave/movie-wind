# movie-wind Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `moviewind` CLI：每日抓取猫眼在映榜 → 累计差分得日票房 → Top10 + ↑↓新 → 映射上市出品方 → Markdown 报告 → 企业微信推送。

**Architecture:** 工程骨架与包边界对齐 `game-wind`（`cmd/moviewind` + `internal/{config,fetch,store,mapping,analyze,report,notify}`）。业务差异集中在 fetch（猫眼）、analyze（日票房 Top10）、mapping（`movies.yaml` 多公司/role）、report（四段正文）。无内置定时；外置 cron 调 `run`。

**Tech Stack:** Go 1.22、`gopkg.in/yaml.v3`、标准库 HTTP/JSON；质量门禁同 game-wind（`gofmt` / `vet` / `golangci-lint` / `go test`）。

参考规格：`docs/superpowers/specs/2026-08-21-movie-wind-design.md`。  
参考实现：`../game-wind/`（同机路径 `/home/chenyinbo/personal/project/game-wind`）。

**金额单位约定（全项目）：** 累计票房与日票房一律用 `float64`，单位 **万元**。`ParseSumBox`：`12.97亿`→`129700`，`8974.0万`→`8974`，纯数字按「元」÷10000。

---

## File map

| Path | Responsibility |
|------|----------------|
| `go.mod` / `go.sum` | module `movie-wind` |
| `.golangci.yml` | 从 game-wind 复制 |
| `AGENTS.md` + `CLAUDE.md`→symlink | 改编自 game-wind（模块名/测试清单改为本仓库） |
| `LICENSE` | MIT，Copyright (c) 2026 chenyinbo |
| `.gitignore` | 对齐 game-wind + 已有探测文件规则 |
| `config.yaml` | `top_n` / `summary_rank_move` / `notify` |
| `cmd/moviewind/main.go` | CLI：`run`/`fetch`/`report`/`list` |
| `internal/config/config.go` | 加载 yaml + 默认值 |
| `internal/fetch/{fetch.go,fetch_test.go}` | 猫眼请求、解析、累计票房 |
| `internal/store/store.go` | 快照 JSON、`state.json`、`LatestBefore` |
| `internal/mapping/{mapping.go,mapping_test.go}` | `movies.yaml`，`MatchAll` |
| `internal/analyze/{analyze.go,analyze_test.go}` | 差分、Top10、标记、总结素材 |
| `internal/report/{report.go,report_test.go}` | Markdown 四段 |
| `internal/notify/{notify.go,notify_test.go}` | 企微推送（基本复制 game-wind） |
| `data/movies.yaml` | 初始映射（可少量真实条目 + 注释） |
| `data/update_movies_yaml.md` | LLM 维护规范 |
| `README.md` | 中文使用说明 |

---

### Task 1: 工程骨架

**Files:**
- Create: `go.mod`, `.golangci.yml`, `AGENTS.md`, `CLAUDE.md` (symlink), `LICENSE`, `.gitignore`, `config.yaml`
- Create: `cmd/moviewind/main.go`（仅 `usage` + 退出码 2）
- Modify: 合并现有 `.gitignore`（保留 `maoyan-live*.json`）

- [ ] **Step 1: 初始化 module 与质量文件**

```bash
cd /home/chenyinbo/personal/project/movie-wind
source ~/.g/env
go mod init movie-wind
go get gopkg.in/yaml.v3@v3.0.1
cp /home/chenyinbo/personal/project/game-wind/.golangci.yml .
cp /home/chenyinbo/personal/project/game-wind/LICENSE .
```

重写 `.gitignore` 为：

```gitignore
/bin/
*.exe
config.local.yaml
/data/*.json
/data/reports/
/logs/
maoyan-live.json
maoyan-live-parsed.json
.idea/
.vscode/
*.swp
.DS_Store
```

`config.yaml`：

```yaml
top_n: 10
summary_rank_move: 3
notify:
  notify_when_quiet: true
  # webhook_url 放 config.local.yaml
```

`AGENTS.md`：从 game-wind 复制后改：
- 标题/工具名为 movie-wind / moviewind
- 第 3 节测试清单先写「随实现追加」，保留一键校验命令

```bash
ln -sf AGENTS.md CLAUDE.md
```

- [ ] **Step 2: 最小 main**

`cmd/moviewind/main.go`：

```go
package main

import (
	"fmt"
	"os"
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
  --date YYYY-MM-DD            指定日期（run/report）
  --data-dir DIR               数据目录（默认 data/）
  --movies FILE                映射表路径（默认 data/movies.yaml）`)
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s（后续任务接线）\n", args[0])
		usage()
		return 2
	}
}
```

- [ ] **Step 3: 校验骨架**

```bash
source ~/.g/env && gofmt -w cmd && go build -o bin/moviewind ./cmd/moviewind && go vet ./... && golangci-lint run ./...
```

Expected: 构建成功；lint 0 issues。

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum .golangci.yml AGENTS.md CLAUDE.md LICENSE .gitignore config.yaml cmd/moviewind/main.go
git commit -m "$(cat <<'EOF'
[chore](cli): scaffold movie-wind Go project aligned with game-wind

EOF
)"
```

若环境 git 过旧不支持 Cursor 注入的 `--trailer`，用 `git write-tree` / `git commit-tree` 提交（同本仓库已有提交方式）。

---

### Task 2: config 包

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: 失败测试**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsAndOverride(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config.yaml")
	local := filepath.Join(dir, "config.local.yaml")
	if err := os.WriteFile(base, []byte("top_n: 10\nsummary_rank_move: 3\nnotify:\n  notify_when_quiet: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("notify:\n  webhook_url: \"https://example.com/hook\"\n  notify_when_quiet: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(base, local)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TopN != 10 || cfg.SummaryRankMove != 3 {
		t.Fatalf("defaults: %+v", cfg)
	}
	if cfg.Notify.WebhookURL != "https://example.com/hook" {
		t.Fatalf("webhook: %q", cfg.Notify.WebhookURL)
	}
	if cfg.Notify.NotifyWhenQuiet {
		t.Fatal("local should override quiet to false")
	}
}

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TopN != 10 || cfg.SummaryRankMove != 3 || !cfg.Notify.NotifyWhenQuiet {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
source ~/.g/env && go test ./internal/config/
```

Expected: FAIL（包不存在或 `Load` 未定义）。

- [ ] **Step 3: 实现**

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Notify struct {
	WebhookURL      string `yaml:"webhook_url"`
	NotifyWhenQuiet bool   `yaml:"notify_when_quiet"`
}

type Config struct {
	TopN            int    `yaml:"top_n"`
	SummaryRankMove int    `yaml:"summary_rank_move"`
	Notify          Notify `yaml:"notify"`
}

func Load(paths ...string) (*Config, error) {
	cfg := &Config{
		TopN:            10,
		SummaryRankMove: 3,
		Notify:          Notify{NotifyWhenQuiet: true},
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("读取配置 %s: %w", p, err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("解析配置 %s: %w", p, err)
		}
	}
	if cfg.TopN <= 0 {
		cfg.TopN = 10
	}
	if cfg.SummaryRankMove <= 0 {
		cfg.SummaryRankMove = 3
	}
	return cfg, nil
}
```

注意：`yaml.Unmarshal` 对 bool 零值无法区分「未写」与 `false`。本地覆盖 `notify_when_quiet: false` 时，后文件整体覆盖 Notify 结构体字段——与 game-wind 相同，依赖后者完整写出要覆盖的键。测试已覆盖。

- [ ] **Step 4: 测试通过并 commit**

```bash
source ~/.g/env && gofmt -w internal/config && go test ./internal/config/ && golangci-lint run ./internal/config/
git add internal/config config.yaml && git commit -m "$(cat <<'EOF'
[feat](config): load top_n, summary_rank_move, and notify settings

EOF
)"
```

---

### Task 3: fetch — ParseSumBox + 解析猫眼 JSON

**Files:**
- Create: `internal/fetch/fetch.go`
- Create: `internal/fetch/fetch_test.go`

- [ ] **Step 1: 失败测试（解析，不打网）**

```go
package fetch

import "testing"

func TestParseSumBox(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"12.97亿", 129700},
		{"8974.0万", 8974},
		{"3526", 0.3526},
		{"", 0},
	}
	for _, c := range cases {
		got, err := ParseSumBox(c.in)
		if c.in == "" {
			if err == nil {
				t.Fatal("empty should error")
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if got < c.want-0.001 || got > c.want+0.001 {
			t.Fatalf("%s: got %v want %v", c.in, got, c.want)
		}
	}
}

func TestParseDashboard(t *testing.T) {
	raw := []byte(`{"movieList":{"list":[{"avgSeatView":"2.1%","avgShowView":"3.1","boxRate":"31.5%","movieInfo":{"movieId":1462628,"movieName":"欢迎来龙餐馆","releaseInfo":"上映11天"},"showCount":143734,"showCountRate":"36.5%","sumBoxDesc":"12.97亿","sumSplitBoxDesc":"11.55亿"}]}}`)
	films, err := ParseDashboard(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(films) != 1 {
		t.Fatalf("len=%d", len(films))
	}
	f := films[0]
	if f.MovieID != 1462628 || f.Name != "欢迎来龙餐馆" {
		t.Fatalf("%+v", f)
	}
	if f.SumBoxWan < 129699 || f.SumBoxWan > 129701 {
		t.Fatalf("sum=%v", f.SumBoxWan)
	}
	if f.BoxRate != "31.5%" || f.ShowCount != 143734 {
		t.Fatalf("%+v", f)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
source ~/.g/env && go test ./internal/fetch/
```

- [ ] **Step 3: 实现类型与解析**

`internal/fetch/fetch.go` 核心：

```go
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
			MovieID: it.MovieInfo.MovieID, Name: it.MovieInfo.MovieName,
			ReleaseInfo: it.MovieInfo.ReleaseInfo, SumBoxDesc: it.SumBoxDesc,
			SumBoxWan: wan, BoxRate: it.BoxRate, ShowCount: it.ShowCount,
			ShowCountRate: it.ShowCountRate, AvgShowView: it.AvgShowView,
			AvgSeatView: it.AvgSeatView,
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
```

- [ ] **Step 4: 测试通过并 commit**

```bash
source ~/.g/env && gofmt -w internal/fetch && go test ./internal/fetch/ && golangci-lint run ./internal/fetch/
git add internal/fetch && git commit -m "$(cat <<'EOF'
[feat](fetch): parse Maoyan dashboard JSON and cumulative box office

EOF
)"
```

---

### Task 4: store 快照与 state

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: 失败测试**

```go
package store

import (
	"path/filepath"
	"testing"
	"time"

	"movie-wind/internal/fetch"
)

func TestSaveLoadLatestBefore(t *testing.T) {
	dir := t.TempDir()
	s1 := &Snapshot{
		Date: "2026-08-20", FetchedAt: time.Now(),
		Films: []fetch.Film{{MovieID: 1, Name: "A", SumBoxWan: 100}},
	}
	s2 := &Snapshot{
		Date: "2026-08-21", FetchedAt: time.Now(),
		Films: []fetch.Film{{MovieID: 1, Name: "A", SumBoxWan: 150}},
	}
	if err := s1.Save(dir); err != nil {
		t.Fatal(err)
	}
	if err := s2.Save(dir); err != nil {
		t.Fatal(err)
	}
	prev, err := LatestBefore(dir, "2026-08-21")
	if err != nil || prev == nil || prev.Date != "2026-08-20" {
		t.Fatalf("prev=%v err=%v", prev, err)
	}
	got, err := Load(dir, "2026-08-21")
	if err != nil || got.Films[0].SumBoxWan != 150 {
		t.Fatalf("%+v %v", got, err)
	}
	// state.json 不应被 ListDates 当成快照
	st := State{LastSnapshotDate: "2026-08-21", LastNotifiedHash: "abc"}
	if err := SaveState(dir, st); err != nil {
		t.Fatal(err)
	}
	dates, err := ListDates(dir)
	if err != nil || len(dates) != 2 {
		t.Fatalf("dates=%v err=%v", dates, err)
	}
	_ = filepath.Join // keep import if needed
}
```

- [ ] **Step 2: 实现**

对齐 game-wind `store`：`Snapshot{Date, FetchedAt, Films []fetch.Film, FetchError string}`；`Save`/`Load`/`ListDates`（仅 `YYYY-MM-DD.json`，长度与后缀校验）；`LatestBefore`；`State`/`LoadState`/`SaveState`。

`ListDates` 必须跳过 `state.json`（已有长度校验即可）。

- [ ] **Step 3: 测试 + commit**

```bash
source ~/.g/env && gofmt -w internal/store && go test ./internal/store/ && golangci-lint run ./internal/store/
git add internal/store && git commit -m "$(cat <<'EOF'
[feat](store): snapshot JSON and notify state persistence

EOF
)"
```

---

### Task 5: mapping — movies.yaml

**Files:**
- Create: `internal/mapping/mapping.go`
- Create: `internal/mapping/mapping_test.go`
- Create: `data/movies.yaml`（最少 1～2 家公司样例，可先用探测到的真实 movieId）

- [ ] **Step 1: 失败测试**

```go
package mapping

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchAllByIDAndName(t *testing.T) {
	yamlContent := `
companies:
  - company: 光线传媒
    code: "300251.SZ"
    market: A股
    films:
      - ids: [1462628]
        names: [欢迎来龙餐馆]
        role: producer
        core: true
  - company: 示例发行
    code: "600000.SH"
    market: A股
    films:
      - ids: [1462628]
        names: [欢迎来龙餐馆]
        role: distributor
        note: 重要发行示例
`
	p := filepath.Join(t.TempDir(), "movies.yaml")
	if err := os.WriteFile(p, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	tab, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	ms := tab.MatchAll(1462628, "欢迎来龙餐馆")
	if len(ms) != 2 {
		t.Fatalf("want 2 companies, got %+v", ms)
	}
	if ms[0].Role != "producer" || !ms[0].Core {
		t.Fatalf("%+v", ms[0])
	}
	ms2 := tab.MatchAll(0, "欢迎来龙餐馆")
	if len(ms2) != 2 {
		t.Fatalf("name match: %+v", ms2)
	}
	if len(tab.Companies) != 2 {
		t.Fatalf("Companies=%d", len(tab.Companies))
	}
}

func TestLoadRequiresCompany(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.yaml")
	_ = os.WriteFile(p, []byte("companies:\n  - code: \"1\"\n"), 0o644)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: 实现要点**

```go
type Film struct {
	IDs   []int64  `yaml:"ids"`
	Names []string `yaml:"names"`
	Role  string   `yaml:"role"` // producer | distributor
	Core  bool     `yaml:"core"`
	Note  string   `yaml:"note,omitempty"`
}

type Company struct {
	Company string `yaml:"company"`
	Code    string `yaml:"code"`
	Market  string `yaml:"market"`
	Films   []Film `yaml:"films"`
}

type Matched struct {
	Company string
	Code    string
	Market  string
	Film    string // names[0]
	Role    string
	Core    bool
	Note    string
}

// MatchAll：按 yaml 公司顺序，收集所有命中（id 优先于 name；同一公司同一片只一条）。
func (t *Table) MatchAll(movieID int64, name string) []*Matched
```

默认 `role` 空则视为 `producer`。

- [ ] **Step 3: 测试 + 样例 yaml + commit**

`data/movies.yaml` 先放注释头 + 至少一条真实探测片（如欢迎来龙餐馆）占位；归属不确定时用 `note: 待核实`，实现报告「待补全」逻辑不受影响。

```bash
source ~/.g/env && gofmt -w internal/mapping && go test ./internal/mapping/
git add internal/mapping data/movies.yaml && git commit -m "$(cat <<'EOF'
[feat](mapping): load movies.yaml and MatchAll by movieId/name

EOF
)"
```

---

### Task 6: analyze — 日票房 Top10 与标记

**Files:**
- Create: `internal/analyze/analyze.go`
- Create: `internal/analyze/analyze_test.go`

- [ ] **Step 1: 写失败测试（四个用例）**

规则（写进 `analyze.go` 文件头注释，测试必须遵守）：

1. `daily(id) = cur.SumBoxWan - prev.SumBoxWan`；缺 prev 同 id 或 daily 非有限 → 不参与 Top。
2. 排序：daily 降序 → SumBoxWan 降序 → MovieID 升序；取前 `TopN`。
3. 上次 Top10：对 `prev` 相对 `prevPrev` 同样计算；`prevPrev == nil` 则上次 Top 为空，本次均为 `新`。
4. `prev == nil` → `Baseline=true`，不做 Top 对比。
5. Mark：不在上次 Top → `新`；名次升 `↑N`；降 `↓N`；同 `—`。
6. `MatchAll` 填 Companies；空则进 Unmapped。
7. Summary：新进；`|Δrank| >= SummaryRankMove`；多公司；core 优先排序。

签名：`func Analyze(cur, prev, prevPrev *store.Snapshot, tab *mapping.Table, opt Options) *Result`

在 `analyze_test.go` 实现：

- `TestBaseline`：prev=nil → Baseline
- `TestTop10OrderAndMarks`：prev+prevPrev 齐全，验证第 1 名 movieId 与 `↑`/`新`/`—`
- `TestUnmapped`：Top 中无映射进 Unmapped
- `TestSummaryNewAndMove`：新进与名次变动进入 Summary

辅助：用 `t.TempDir()` 写一小段 `movies.yaml` 再 `mapping.Load`。

类型：

```go
type Options struct {
	TopN            int
	SummaryRankMove int
}

type TopItem struct {
	Rank      int
	MovieID   int64
	Name      string
	DailyWan  float64
	SumBoxWan float64
	BoxRate   string
	Mark      string // 新 | ↑N | ↓N | —
	Companies []*mapping.Matched
}

type SummaryItem struct {
	Kind string // new | move | multi
	Text string
	Core bool
}

type Result struct {
	Baseline bool
	Top      []TopItem
	Unmapped []TopItem
	Summary  []SummaryItem
	Notes    []string
}
```

按公司视图由 `report` 用 `tab.Companies` + `Result.Top` 组装。

- [ ] **Step 2: 跑测试确认失败**

```bash
source ~/.g/env && go test ./internal/analyze/
```

Expected: FAIL（包或符号不存在）。

- [ ] **Step 3: 实现 `analyze.go` 使四用例通过**

- [ ] **Step 4: 测试全绿 + commit**

```bash
source ~/.g/env && gofmt -w internal/analyze && go test ./internal/analyze/ && golangci-lint run ./internal/analyze/
git add internal/analyze && git commit -m "$(cat <<'EOF'
[feat](analyze): daily box-office Top10 with rank marks and summary

EOF
)"
```

---

### Task 7: report Markdown

**Files:**
- Create: `internal/report/report.go`
- Create: `internal/report/report_test.go`

- [ ] **Step 1: 失败测试**

断言 `Build` 输出包含：

- `# 电影公司观察 · 2026-08-21`
- `## 一、日票房 Top10`
- `## 二、影视公司`
- `## 三、总结`
- `## 四、待补全`
- 公司段对 yaml 中无 Top 关联的公司出现「今日 Top10 无关联影片」
- Top 行含 `(↑1)` / `(新)` 形式

```go
func TestBuildSections(t *testing.T) {
	// 构造 tab（2 公司）、Result（1 条 Top 映射公司A，1 条 Unmapped）
	md := Build("2026-08-21", res, tab, "", false)
	for _, want := range []string{"## 一、日票房 Top10", "## 二、影视公司", "## 三、总结", "## 四、待补全", "今日 Top10 无关联影片"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
}
```

- [ ] **Step 2: 实现 `Build(date string, res *analyze.Result, tab *mapping.Table, fetchErr string, baseline bool) string`**

baseline 时输出简短「基线已建立」正文（供 notify）。

格式化日票房：保留 1 位小数 + `万`。

- [ ] **Step 3: 测试 + commit**

```bash
git add internal/report && git commit -m "$(cat <<'EOF'
[feat](report): render Top10, companies, summary, and unmapped sections

EOF
)"
```

---

### Task 8: notify（对齐 game-wind）

**Files:**
- Create: `internal/notify/notify.go`
- Create: `internal/notify/notify_test.go`

- [ ] **Step 1:** 从 game-wind 复制 `notify.go` / 相关测试，把 import 改为 `movie-wind`；保留 `BuildMessages`、`SendMarkdown`、`MaxBytes=4000`。

- [ ] **Step 2:**

```bash
source ~/.g/env && go test ./internal/notify/
git add internal/notify && git commit -m "$(cat <<'EOF'
[feat](notify): WeCom markdown send and split aligned with game-wind

EOF
)"
```

---

### Task 9: CLI 接线 `run`/`fetch`/`report`/`list`

**Files:**
- Modify: `cmd/moviewind/main.go`（参照 game-wind `main.go` 结构）

- [ ] **Step 1: 实现选项解析与命令**

常量：

```go
const (
	configPath     = "config.yaml"
	localCfgPath   = "config.local.yaml"
	moviesFile     = "data/movies.yaml"
	defaultDataDir = "data"
)
```

`cmdFetch`：`Fetch` → `Snapshot{Date, FetchedAt, Films}` → `Save`；失败写 `FetchError` 并返回非 0。

`cmdList`：打印 `ListDates`。

`cmdReport` / `cmdRun` 共用 `buildAndMaybeNotify`：

1. `Load` cur；`LatestBefore` → prev；若 prev≠nil 再 `LatestBefore(prev.Date)` → prevPrev  
2. `Analyze(cur, prev, prevPrev, tab, opts)`  
3. `report.Build`  
4. 可选写入 `data/reports/YYYY-MM-DD.md`  
5. 非 dry-run：指纹 `sha256(body)`；与 `state.LastNotifiedHash` 同日相同则跳过；`notify_when_quiet` 已默认 true，仍推完整报告；baseline 推确认消息  
6. `SaveState`

`cmdRun`：先 fetch（除非 `--date` 指向已有快照且你选择跳过——对齐 game-wind：`--date` 时用已有快照不重新抓，或仍允许重抓覆盖；**采用与 game-wind 相同：run 总是先抓取写入当日，再用该日分析**；若指定 `--date` 则只分析该日已有文件不抓取）。

对齐 game-wind 行为请直接读 `/home/chenyinbo/personal/project/game-wind/cmd/gamewind/main.go` 的 `cmdRun`/`cmdReport` 并改编。

- [ ] **Step 2: 编译与本地 dry-run**

```bash
source ~/.g/env && gofmt -w cmd internal && go build -o bin/moviewind ./cmd/moviewind
./bin/moviewind fetch --data-dir /tmp/mw-data
./bin/moviewind list --data-dir /tmp/mw-data
./bin/moviewind report --dry-run --data-dir /tmp/mw-data
```

Expected: 抓取成功则生成 JSON；首次 report 为基线文案；第二次不同日才能出 Top10（可用手工复制改日期测）。

- [ ] **Step 3: 全量检查 + commit**

```bash
source ~/.g/env && gofmt -w cmd internal && go build ./... && go vet ./... && golangci-lint run ./... && go test ./...
git add cmd/moviewind && git commit -m "$(cat <<'EOF'
[feat](cli): wire run/fetch/report/list end-to-end pipeline

EOF
)"
```

---

### Task 10: README + update_movies_yaml.md + 样例完善

**Files:**
- Create: `README.md`
- Create: `data/update_movies_yaml.md`
- Modify: `data/movies.yaml`（补充注释与维护说明）
- Modify: `AGENTS.md` 第 3 节测试清单为真实用例表

- [ ] **Step 1: README** 结构对齐 game-wind：功能、快速开始、命令、配置、cron 示例（`0 8 * * *`）、数据源说明、差分口径、FAQ。

- [ ] **Step 2: `update_movies_yaml.md`** 改编自 game-wind `update_game_yaml.md`：只收录 A 股+港股通；role 规则；重要发行须 note；匹配优先级 ids>names；读最新 `data/*.json` Top 待补全。

- [ ] **Step 3: Commit**

```bash
git add README.md data/update_movies_yaml.md data/movies.yaml AGENTS.md
git commit -m "$(cat <<'EOF'
[docs](cli): add README, movies.yaml maintainer prompt, and test inventory

EOF
)"
```

---

### Task 11: 规格对照自检

- [ ] **Step 1:** 打开 `docs/superpowers/specs/2026-08-21-movie-wind-design.md`，逐条确认已实现：猫眼源、累计差分、yaml 出品/重要发行、Top10↑↓新、四段报告、全公司段、安静日默认推送、无内置定时、Go 工程对齐。

- [ ] **Step 2:** 再跑一遍：

```bash
source ~/.g/env && gofmt -w cmd internal && go build ./... && go vet ./... && golangci-lint run ./... && go test ./...
```

- [ ] **Step 3:** 若有缺口，开小提交修补；无则本任务不另提交。

---

## Spec coverage checklist（计划自检）

| 规格项 | Task |
|--------|------|
| 猫眼 fetch + 重试 | 3, 9 |
| 累计差分日票房（万元） | 3, 6 |
| movies.yaml 出品/重要发行 | 5 |
| Top10 + ↑↓新 | 6 |
| 报告四段 + 全公司 | 7 |
| notify_when_quiet 默认 true | 2, 9 |
| 外置 cron / 无 scheduler | 9, 10 |
| 对齐 game-wind 工程 | 1, 8, 10 |
| 基线首次运行 | 6, 9 |

## Placeholder scan

计划中禁止残留 TBD；金额单位、Analyze 签名、标记规则已写死。Task 6 要求四个命名测试：`TestBaseline` / `TestTop10OrderAndMarks` / `TestUnmapped` / `TestSummaryNewAndMove`。
