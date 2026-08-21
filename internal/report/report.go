// Package report 生成 Markdown 日报。
package report

import (
	"fmt"
	"strings"

	"movie-wind/internal/analyze"
	"movie-wind/internal/mapping"
)

// Build 生成完整报告；baseline 时输出基线确认文案。
func Build(date string, res *analyze.Result, tab *mapping.Table, fetchErr string, baseline bool) string {
	if baseline || (res != nil && res.Baseline) {
		var b strings.Builder
		fmt.Fprintf(&b, "# 电影公司观察 · %s\n", date)
		b.WriteString("> 数据源：猫眼在映榜 · 日票房=累计差分 · 与最近一次快照对比\n")
		if fetchErr != "" {
			fmt.Fprintf(&b, "> 抓取状态：%s\n", fetchErr)
		}
		b.WriteString("\n基线已建立：已保存今日快照。下次运行将计算日票房 Top10 并对比名次变化。\n")
		return b.String()
	}
	if res == nil {
		res = &analyze.Result{}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# 电影公司观察 · %s\n", date)
	b.WriteString("> 数据源：猫眼在映榜 · 日票房=累计差分 · 与最近一次快照对比\n")
	if fetchErr != "" {
		fmt.Fprintf(&b, "> 抓取状态：%s\n", fetchErr)
	}
	b.WriteString("\n")

	b.WriteString("## 一、日票房 Top10\n\n")
	if len(res.Top) == 0 {
		b.WriteString("（无可用日票房差分，或影片不足）\n\n")
	} else {
		for _, it := range res.Top {
			fmt.Fprintf(&b, "%d. 《%s》(%d) 第%d(%s) · 日票房 %s · 累计 %s",
				it.Rank, it.Name, it.MovieID, it.Rank, it.Mark, fmtWan(it.DailyWan), fmtWan(it.SumBoxWan))
			if it.BoxRate != "" {
				fmt.Fprintf(&b, " · 占比 %s", it.BoxRate)
			}
			if len(it.Companies) > 0 {
				fmt.Fprintf(&b, " · 关联：%s", formatCompanies(it.Companies))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## 二、影视公司\n\n")
	if tab == nil || len(tab.Companies) == 0 {
		b.WriteString("（映射表为空）\n\n")
	} else {
		byCompany := map[string][]analyze.TopItem{}
		for _, it := range res.Top {
			for _, m := range it.Companies {
				key := m.Company + "\x00" + m.Code
				byCompany[key] = append(byCompany[key], it)
			}
		}
		for _, c := range tab.Companies {
			key := c.Company + "\x00" + c.Code
			fmt.Fprintf(&b, "### %s · %s\n", c.Company, c.Code)
			items := byCompany[key]
			if len(items) == 0 {
				b.WriteString("- 今日 Top10 无关联影片\n\n")
				continue
			}
			seen := map[int64]bool{}
			for _, it := range items {
				if seen[it.MovieID] {
					continue
				}
				seen[it.MovieID] = true
				role := roleFor(&it, c.Company, c.Code)
				fmt.Fprintf(&b, "- 《%s》%s · 日票房第%d(%s) · %s\n",
					it.Name, role, it.Rank, it.Mark, fmtWan(it.DailyWan))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("## 三、总结\n\n")
	if len(res.Summary) == 0 {
		b.WriteString("Top10 成员与名次相对稳定\n\n")
	} else {
		b.WriteString("值得关注：\n")
		for _, s := range res.Summary {
			fmt.Fprintf(&b, "- %s\n", s.Text)
		}
		b.WriteString("\n")
	}
	for _, n := range res.Notes {
		fmt.Fprintf(&b, "> 备注：%s\n", n)
	}

	b.WriteString("\n## 四、待补全\n\n")
	if len(res.Unmapped) == 0 {
		b.WriteString("（Top10 均已映射）\n")
	} else {
		for _, it := range res.Unmapped {
			fmt.Fprintf(&b, "- 第%d(%s) 《%s》 movieId=%d · 日票房 %s（请核实出品方是否为上市标的）\n",
				it.Rank, it.Mark, it.Name, it.MovieID, fmtWan(it.DailyWan))
		}
	}
	return b.String()
}

func fmtWan(v float64) string {
	return fmt.Sprintf("%.1f万", v)
}

func formatCompanies(ms []*mapping.Matched) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		role := "出品"
		if m.Role == "distributor" {
			role = "发行"
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", m.Company, role))
	}
	return strings.Join(parts, "、")
}

func roleFor(it *analyze.TopItem, company, code string) string {
	for _, m := range it.Companies {
		if m.Company == company && m.Code == code {
			if m.Role == "distributor" {
				return "发行"
			}
			return "出品"
		}
	}
	return "出品"
}
