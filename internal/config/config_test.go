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
