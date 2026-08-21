// Package analyze 根据累计票房差分计算日票房 TopN 与名次标记。
//
// 规则：
//  1. daily(id) = cur.SumBoxWan - prev.SumBoxWan；缺 prev 同 id 或 daily 非有限 → 不参与 Top。
//  2. 排序：daily 降序 → SumBoxWan 降序 → MovieID 升序；取前 TopN。
//  3. 上次 Top：对 prev 相对 prevPrev 同样计算；prevPrev==nil 则上次为空，本次均为「新」。
//  4. prev==nil → Baseline=true。
//  5. Mark：不在上次 Top → 新；升 ↑N；降 ↓N；同 —。
package analyze

import (
	"fmt"
	"math"
	"sort"

	"movie-wind/internal/fetch"
	"movie-wind/internal/mapping"
	"movie-wind/internal/store"
)

// Options 分析参数。
type Options struct {
	TopN            int
	SummaryRankMove int
}

// TopItem 日票房榜一条。
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

// SummaryItem 总结条目。
type SummaryItem struct {
	Kind string // new | move | multi
	Text string
	Core bool
}

// Result 分析结果。
type Result struct {
	Baseline bool
	Top      []TopItem
	Unmapped []TopItem
	Summary  []SummaryItem
	Notes    []string
}

type ranked struct {
	film  fetch.Film
	daily float64
}

// Analyze 对比快照，产出日票房 TopN。
func Analyze(cur, prev, prevPrev *store.Snapshot, tab *mapping.Table, opt Options) *Result {
	if opt.TopN <= 0 {
		opt.TopN = 10
	}
	if opt.SummaryRankMove <= 0 {
		opt.SummaryRankMove = 3
	}
	res := &Result{}
	if cur == nil || prev == nil {
		res.Baseline = true
		return res
	}

	curTop := rankDaily(cur, prev, opt.TopN, res)
	prevRank := map[int64]int{}
	if prevPrev != nil {
		prevTop := rankDaily(prev, prevPrev, opt.TopN, nil)
		for i, it := range prevTop {
			prevRank[it.film.MovieID] = i + 1
		}
	}

	for i, r := range curTop {
		item := TopItem{
			Rank:      i + 1,
			MovieID:   r.film.MovieID,
			Name:      r.film.Name,
			DailyWan:  r.daily,
			SumBoxWan: r.film.SumBoxWan,
			BoxRate:   r.film.BoxRate,
			Mark:      markForID(r.film.MovieID, i+1, prevRank),
		}
		if tab != nil {
			item.Companies = tab.MatchAll(r.film.MovieID, r.film.Name)
		}
		if len(item.Companies) == 0 {
			res.Unmapped = append(res.Unmapped, item)
		}
		res.Top = append(res.Top, item)
	}
	res.Summary = buildSummary(res.Top, opt.SummaryRankMove)
	return res
}

func markForID(movieID int64, rank int, prevRank map[int64]int) string {
	old, ok := prevRank[movieID]
	if !ok {
		return "新"
	}
	if old == rank {
		return "—"
	}
	if rank < old {
		return fmt.Sprintf("↑%d", old-rank)
	}
	return fmt.Sprintf("↓%d", rank-old)
}

func rankDaily(cur, prev *store.Snapshot, topN int, res *Result) []ranked {
	prevSum := map[int64]float64{}
	for _, f := range prev.Films {
		prevSum[f.MovieID] = f.SumBoxWan
	}
	var list []ranked
	for _, f := range cur.Films {
		ps, ok := prevSum[f.MovieID]
		if !ok {
			continue
		}
		daily := f.SumBoxWan - ps
		if math.IsNaN(daily) || math.IsInf(daily, 0) {
			continue
		}
		if daily < 0 && res != nil {
			res.Notes = append(res.Notes, fmt.Sprintf("%s(%d) 日票房为负 %.1f 万", f.Name, f.MovieID, daily))
		}
		list = append(list, ranked{film: f, daily: daily})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].daily != list[j].daily {
			return list[i].daily > list[j].daily
		}
		if list[i].film.SumBoxWan != list[j].film.SumBoxWan {
			return list[i].film.SumBoxWan > list[j].film.SumBoxWan
		}
		return list[i].film.MovieID < list[j].film.MovieID
	})
	if len(list) > topN {
		list = list[:topN]
	}
	return list
}

func buildSummary(top []TopItem, moveN int) []SummaryItem {
	var out []SummaryItem
	for _, it := range top {
		core := false
		for _, c := range it.Companies {
			if c.Core {
				core = true
				break
			}
		}
		co := companyText(it.Companies)
		switch {
		case it.Mark == "新":
			out = append(out, SummaryItem{
				Kind: "new",
				Core: core,
				Text: fmt.Sprintf("《%s》新进日票房第%d%s", it.Name, it.Rank, co),
			})
		case parseMarkDelta(it.Mark) >= moveN:
			out = append(out, SummaryItem{
				Kind: "move",
				Core: core,
				Text: fmt.Sprintf("《%s》日票房第%d(%s)%s", it.Name, it.Rank, it.Mark, co),
			})
		}
		if len(it.Companies) >= 2 {
			out = append(out, SummaryItem{
				Kind: "multi",
				Core: core,
				Text: fmt.Sprintf("《%s》关联多家：%s", it.Name, joinCompanies(it.Companies)),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Core != out[j].Core {
			return out[i].Core
		}
		return false
	})
	return out
}

func parseMarkDelta(mark string) int {
	r := []rune(mark)
	if len(r) < 2 || (r[0] != '↑' && r[0] != '↓') {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(string(r[1:]), "%d", &n)
	return n
}

func companyText(ms []*mapping.Matched) string {
	if len(ms) == 0 {
		return ""
	}
	return " · " + joinCompanies(ms)
}

func joinCompanies(ms []*mapping.Matched) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		role := "出品"
		if m.Role == "distributor" {
			role = "发行"
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", m.Company, role))
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "、"
		}
		out += p
	}
	return out
}
