package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"movie-wind/internal/analyze"
	"movie-wind/internal/mapping"
)

func TestBuildSections(t *testing.T) {
	yamlContent := `
companies:
  - company: 公司A
    code: "1.SZ"
    market: A股
    films:
      - ids: [1]
        names: [片A]
  - company: 公司B
    code: "2.SZ"
    market: A股
    films: []
`
	p := filepath.Join(t.TempDir(), "m.yaml")
	_ = os.WriteFile(p, []byte(yamlContent), 0o644)
	tab, err := mapping.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	res := &analyze.Result{
		Top: []analyze.TopItem{{
			Rank: 1, MovieID: 1, Name: "片A", DailyWan: 10, SumBoxWan: 100,
			Mark: "↑1", Companies: tab.MatchAll(1, "片A"),
		}},
		Unmapped: []analyze.TopItem{{
			Rank: 2, MovieID: 9, Name: "未映射", DailyWan: 5, Mark: "新",
		}},
		Summary: []analyze.SummaryItem{{Kind: "move", Text: "《片A》日票房第1(↑1)"}},
	}
	md := Build("2026-08-21", res, tab, "", false)
	for _, want := range []string{
		"## 一、日票房 Top10",
		"## 二、影视公司",
		"## 三、总结",
		"## 四、待补全",
		"今日 Top10 无关联影片",
		"(↑1)",
		"(新)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
}

func TestBuildBaseline(t *testing.T) {
	md := Build("2026-08-21", &analyze.Result{Baseline: true}, nil, "", true)
	if !strings.Contains(md, "基线已建立") {
		t.Fatal(md)
	}
}
