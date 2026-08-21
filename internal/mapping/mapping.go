// Package mapping 加载 data/movies.yaml 影片→上市公司映射表并提供匹配。
package mapping

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Film 一部影片在某公司下的映射条目。
type Film struct {
	IDs   []int64  `yaml:"ids"`
	Names []string `yaml:"names"`
	Role  string   `yaml:"role"` // producer | distributor；空视为 producer
	Core  bool     `yaml:"core"`
	Note  string   `yaml:"note,omitempty"`
}

// Company 一家上市公司及其影片列表。
type Company struct {
	Company string `yaml:"company"`
	Code    string `yaml:"code"`
	Market  string `yaml:"market"`
	Films   []Film `yaml:"films"`
}

// Matched 单次匹配结果。
type Matched struct {
	Company string
	Code    string
	Market  string
	Film    string
	Role    string
	Core    bool
	Note    string
}

type tableFile struct {
	Companies []Company `yaml:"companies"`
}

// Table 映射表。
type Table struct {
	byID      map[int64][]*Matched
	byName    map[string][]*Matched
	Companies []Company
}

// Load 从 YAML 文件加载映射表。
func Load(path string) (*Table, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取映射表 %s: %w", path, err)
	}
	var tf tableFile
	if err := yaml.Unmarshal(b, &tf); err != nil {
		return nil, fmt.Errorf("解析映射表 %s: %w", path, err)
	}
	t := &Table{
		byID:      make(map[int64][]*Matched),
		byName:    make(map[string][]*Matched),
		Companies: tf.Companies,
	}
	for _, c := range tf.Companies {
		if c.Company == "" {
			return nil, fmt.Errorf("映射表 %s: 存在缺少 company 的条目", path)
		}
		for _, f := range c.Films {
			role := f.Role
			if role == "" {
				role = "producer"
			}
			name := "(未命名条目)"
			if len(f.Names) > 0 {
				name = f.Names[0]
			}
			m := &Matched{
				Company: c.Company,
				Code:    c.Code,
				Market:  c.Market,
				Film:    name,
				Role:    role,
				Core:    f.Core,
				Note:    f.Note,
			}
			for _, id := range f.IDs {
				t.byID[id] = append(t.byID[id], m)
			}
			for _, n := range f.Names {
				t.byName[n] = append(t.byName[n], m)
			}
		}
	}
	return t, nil
}

// MatchAll 按公司 yaml 顺序收集所有命中：有 id 命中时只用 id；否则用精确名。
// 同一公司同一片只保留一条。
func (t *Table) MatchAll(movieID int64, name string) []*Matched {
	if t == nil {
		return nil
	}
	var raw []*Matched
	if movieID != 0 {
		if ms, ok := t.byID[movieID]; ok {
			raw = ms
		}
	}
	if len(raw) == 0 && name != "" {
		raw = t.byName[name]
	}
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]*Matched, 0, len(raw))
	for _, m := range raw {
		key := m.Company + "\x00" + m.Code + "\x00" + m.Role
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, m)
	}
	return out
}
