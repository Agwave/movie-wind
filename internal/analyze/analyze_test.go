package analyze

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"movie-wind/internal/fetch"
	"movie-wind/internal/mapping"
	"movie-wind/internal/store"
)

func loadTab(t *testing.T, yaml string) *mapping.Table {
	t.Helper()
	p := filepath.Join(t.TempDir(), "movies.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	tab, err := mapping.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func TestBaseline(t *testing.T) {
	cur := &store.Snapshot{Date: "2026-08-21", Films: []fetch.Film{{MovieID: 1, Name: "A", SumBoxWan: 100}}}
	res := Analyze(cur, nil, nil, nil, Options{TopN: 10, SummaryRankMove: 3})
	if !res.Baseline {
		t.Fatal("expected baseline")
	}
	if len(res.Top) != 0 {
		t.Fatalf("top=%+v", res.Top)
	}
}

func TestTop10OrderAndMarks(t *testing.T) {
	// Day1 cumulative
	pp := &store.Snapshot{Date: "2026-08-19", Films: []fetch.Film{
		{MovieID: 1, Name: "A", SumBoxWan: 100},
		{MovieID: 2, Name: "B", SumBoxWan: 100},
		{MovieID: 3, Name: "C", SumBoxWan: 100},
	}}
	// Day2: A daily 50 (rank2), B daily 80 (rank1), C daily 10 (rank3)
	prev := &store.Snapshot{Date: "2026-08-20", Films: []fetch.Film{
		{MovieID: 1, Name: "A", SumBoxWan: 150},
		{MovieID: 2, Name: "B", SumBoxWan: 180},
		{MovieID: 3, Name: "C", SumBoxWan: 110},
	}}
	// Day3: A daily 100 (rank1 ↑1), B daily 20 (rank3 ↓2), C daily 40 (rank2 ↑1), D new but no prev → skip
	cur := &store.Snapshot{Date: "2026-08-21", Films: []fetch.Film{
		{MovieID: 1, Name: "A", SumBoxWan: 250},
		{MovieID: 2, Name: "B", SumBoxWan: 200},
		{MovieID: 3, Name: "C", SumBoxWan: 150},
		{MovieID: 4, Name: "D", SumBoxWan: 999},
	}}
	tab := loadTab(t, `
companies:
  - company: CoA
    code: "1"
    market: A股
    films:
      - ids: [1]
        names: [A]
        role: producer
`)
	res := Analyze(cur, prev, pp, tab, Options{TopN: 10, SummaryRankMove: 1})
	if res.Baseline {
		t.Fatal("not baseline")
	}
	if len(res.Top) != 3 {
		t.Fatalf("want 3 (D skipped), got %+v", res.Top)
	}
	if res.Top[0].MovieID != 1 || res.Top[0].Mark != "↑1" {
		t.Fatalf("rank1=%+v", res.Top[0])
	}
	if res.Top[1].MovieID != 3 || res.Top[1].Mark != "↑1" {
		t.Fatalf("rank2=%+v", res.Top[1])
	}
	if res.Top[2].MovieID != 2 || res.Top[2].Mark != "↓2" {
		t.Fatalf("rank3=%+v", res.Top[2])
	}
}

func TestUnmapped(t *testing.T) {
	pp := &store.Snapshot{Date: "2026-08-19", Films: []fetch.Film{{MovieID: 9, Name: "X", SumBoxWan: 10}}}
	prev := &store.Snapshot{Date: "2026-08-20", Films: []fetch.Film{{MovieID: 9, Name: "X", SumBoxWan: 20}}}
	cur := &store.Snapshot{Date: "2026-08-21", Films: []fetch.Film{{MovieID: 9, Name: "X", SumBoxWan: 50}}}
	tab := loadTab(t, "companies: []\n")
	res := Analyze(cur, prev, pp, tab, Options{TopN: 10, SummaryRankMove: 3})
	if len(res.Unmapped) != 1 || res.Unmapped[0].MovieID != 9 {
		t.Fatalf("%+v", res.Unmapped)
	}
}

func TestSummaryNewAndMove(t *testing.T) {
	pp := &store.Snapshot{Date: "2026-08-19", Films: []fetch.Film{
		{MovieID: 1, Name: "A", SumBoxWan: 100},
		{MovieID: 2, Name: "B", SumBoxWan: 100},
	}}
	prev := &store.Snapshot{Date: "2026-08-20", Films: []fetch.Film{
		{MovieID: 1, Name: "A", SumBoxWan: 200}, // daily 100 → was #1
		{MovieID: 2, Name: "B", SumBoxWan: 150}, // daily 50  → was #2
	}}
	cur := &store.Snapshot{Date: "2026-08-21", Films: []fetch.Film{
		{MovieID: 2, Name: "B", SumBoxWan: 400}, // daily 250 → #1 ↑1
		{MovieID: 1, Name: "A", SumBoxWan: 220}, // daily 20  → #2 ↓1
		{MovieID: 3, Name: "C", SumBoxWan: 100}, // need prev for daily — add to prev
	}}
	// Give C a prev entry so it can enter as 新 relative to prevPrev top
	prev.Films = append(prev.Films, fetch.Film{MovieID: 3, Name: "C", SumBoxWan: 10})
	cur.Films[2].SumBoxWan = 200 // daily 190 → will be high; not in prevPrev top → 新

	tab := loadTab(t, `
companies:
  - company: CoB
    code: "2"
    market: A股
    films:
      - ids: [2]
        names: [B]
        core: true
`)
	res := Analyze(cur, prev, pp, tab, Options{TopN: 10, SummaryRankMove: 1})
	var kinds []string
	for _, s := range res.Summary {
		kinds = append(kinds, s.Kind)
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, "new") && !strings.Contains(joined, "move") {
		t.Fatalf("summary=%+v", res.Summary)
	}
	foundNew := false
	for _, s := range res.Summary {
		if s.Kind == "new" && strings.Contains(s.Text, "C") {
			foundNew = true
		}
	}
	if !foundNew {
		t.Fatalf("want C as 新 in summary: %+v", res.Summary)
	}
}
