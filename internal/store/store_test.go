package store

import (
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
	st := State{LastSnapshotDate: "2026-08-21", LastNotifiedHash: "abc"}
	if err := SaveState(dir, st); err != nil {
		t.Fatal(err)
	}
	dates, err := ListDates(dir)
	if err != nil || len(dates) != 2 {
		t.Fatalf("dates=%v err=%v", dates, err)
	}
	loaded := LoadState(dir)
	if loaded.LastNotifiedHash != "abc" {
		t.Fatalf("%+v", loaded)
	}
}
