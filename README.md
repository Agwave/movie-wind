# movie-wind · A股/港股影视公司观察工具

每天抓取猫眼在映票房，用累计差分得到近似日票房，聚焦 **日票房 Top10**，
映射到白名单上市公司（默认出品；少数重要发行可选），生成 Markdown 报告并推送到企业微信。

> **定位**：只看 **A 股上市公司 + 港股通标的港股** 影视相关标的。未上市片方不进白名单；
> Top10 未映射影片列入报告「待补全」，可用 `data/update_movies_yaml.md` 维护补充。

## 功能特性

- 📡 **每日抓取**：猫眼 `dashboard-ajax/movie`（当前免登录）
- 📊 **日票房**：`本次累计 − 上次快照累计`（万元）；不解码字体加密字段
- 🏆 **只盯 Top10**：相对上次标记 `新` / `↑N` / `↓N` / `—`
- 🗂️ **报告四段**：Top10 → 影视公司（yaml 全部）→ 总结 → 待补全
- 📲 **企业微信**：默认每日推送（`notify_when_quiet: true`）；指纹防重；超长拆条

## 快速开始

```bash
# 1. 环境（Go 1.22，通过 g 管理）
source ~/.g/env
g install 1.22.12 && g use 1.22.12

# 2. 构建
go build -o bin/moviewind ./cmd/moviewind

# 3. 首次运行（建立基线，不推送）
./bin/moviewind run --dry-run

# 4. 配置企微后正式跑
# 写入 config.local.yaml：
# notify:
#   webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
./bin/moviewind run
```

## 命令参考

| 命令 | 说明 |
|---|---|
| `moviewind run [--dry-run] [--date YYYY-MM-DD]` | 无 `--date` 时抓取+分析+报告+推送；有 `--date` 时只用已有快照 |
| `moviewind fetch` | 只抓取入库 |
| `moviewind report [--date] [--dry-run]` | 用已有快照生成报告/推送 |
| `moviewind list` | 列出已有快照 |
| `--data-dir DIR` | 数据目录（默认 `data/`） |
| `--movies FILE` | 映射表路径（默认 `data/movies.yaml`） |

## 配置

### 企业微信

写入 `config.local.yaml`（gitignore）：

```yaml
notify:
  webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
```

### config.yaml

```yaml
top_n: 10
summary_rank_move: 3          # 总结段：名次变动 ≥ N 才写入
notify:
  notify_when_quiet: true     # 无总结变化时是否仍推完整报告（默认 true）
```

### 映射表 data/movies.yaml

匹配优先级：**movieId > 精确片名**。`role` 默认 `producer`；重要发行用 `distributor` 并写 `note`。

## 定时任务

工具**不内置定时**。建议每天 8:00：

```bash
go build -o bin/moviewind ./cmd/moviewind
mkdir -p logs
crontab -e
# 每天 8:00
0 8 * * * cd /path/to/movie-wind && ./bin/moviewind run >> logs/cron.log 2>&1
```

## 数据源

- `https://piaofang.maoyan.com/dashboard-ajax/movie`
- 日票房 = 累计差分，与官方日结可能有偏差；报告表述为「与最近一次快照对比」

## 许可证

[MIT](LICENSE) © 2026 chenyinbo
