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
