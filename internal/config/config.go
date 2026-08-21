package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Notify 通知配置。
type Notify struct {
	WebhookURL      string `yaml:"webhook_url"`
	NotifyWhenQuiet bool   `yaml:"notify_when_quiet"`
}

// Config 总配置。
type Config struct {
	TopN            int    `yaml:"top_n"`
	SummaryRankMove int    `yaml:"summary_rank_move"`
	Notify          Notify `yaml:"notify"`
}

// Load 按顺序加载多个 YAML 配置文件，后者覆盖前者的同名键，并填充默认值。
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
