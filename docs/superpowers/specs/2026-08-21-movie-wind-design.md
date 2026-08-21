# movie-wind 设计规格

日期：2026-08-21 · 状态：已确认

> 面向 A 股 / 港股通影视相关投资者的观察工具。形态对齐 `game-wind`（抓取 → 快照 → 分析 → Markdown 报告 → 企业微信），分析逻辑按国内日票房 Top10，不做游戏榜那套大幅跳变阈值。

## 1. 背景与目标

`movie-wind` 每日（建议上午 8:00，外置 cron）抓取国内在映票房，用累计票房差分得到近似日票房，聚焦 **日票房前十** 影片，映射到白名单上市公司（默认出品方；少数重要发行可选），生成报告并推送。

成功标准：

- 盘前能稳定看到「日票房 Top10 + 关联哪些上市公司」
- 相对上次快照能看出名次 ↑↓ 与是否新进前十
- 未映射的 Top10 片子可驱动维护 `movies.yaml`

非目标（MVP）：

- 不解码猫眼字体加密的当日票房字段
- 不做盘中小时级告警（可二期）
- 不做院线、普通联合发行、未上市片方的利好叙事
- 不内置进程内定时器

## 2. 数据源与日票房口径

### 2.1 主数据源

- URL：`GET https://piaofang.maoyan.com/dashboard-ajax/movie`
- 当前：无需登录；建议带常规 `User-Agent` + `Referer: https://piaofang.maoyan.com/dashboard`
- 落盘明文：`movieId`、片名、累计票房、票房占比、场次、排片占比、场均人次、上座率、上映信息等
- **不使用** `boxSplitUnit.num`（字体加密）

失败策略：重试 2 次；仍失败则当日不计算差分、报告标注抓取失败，不推半截「假 Top10」。

可选校验（不进主账本）：第三方 Top10 API（如 apizero）仅作日志交叉核对；无稳定 `movieId`、覆盖窄。

### 2.2 日票房

```
日票房(movieId) = 本次快照累计票房 − 上一份成功快照累计票房
```

- 报告 / 快照日期 = **运行日历日** `YYYY-MM-DD`（不强调「营业日昨/今」文案，对齐 game-wind）
- 建议 cron：每天 `08:00` 执行一次 `moviewind run`（结算与推送默认同一次；命令上仍拆开以便以后分时）
- 首次运行：只建基线；可推送一条确认消息；不做 ↑↓/新 对比
- 差分异常（负值等）：写入抓取/分析备注，不作为利好解读

累计票房解析：将 `sumBoxDesc` 一类中文单位（亿/万）统一为同一数值单位（实现时固定为「万元」或「元」，全项目一致）。

### 2.3 定时归属

- **不在程序内做 scheduler**
- 由 crontab / systemd timer 等外部触发
- 程序提供：`run` / `fetch` / `report` / `list`

## 3. 影片 → 上市公司映射

### 3.1 公司宇宙

只收录：

- A 股上市公司
- 港股通标的港股公司

未上市片方、纯外企、已退市：不进白名单；Top10 未映射进「待补全」。

### 3.2 角色规则

- **默认**：`producer`（出品 / 联合出品）
- **可选**：`distributor`（发行），仅当「重要发行」时写入，且 `note` 写明依据（官宣 / 招股书 / 主流财经报道等）
- 普通联合发行、院线排片、纯宣发代理：默认不收录

### 3.3 文件与匹配

- `data/movies.yaml`：公司 → 影片列表
- `data/update_movies_yaml.md`：给大模型的维护规范（对齐 game-wind 的 `update_game_yaml.md`）

匹配优先级：`ids`（猫眼 `movieId`）> `names` 精确匹配。

示意：

```yaml
companies:
  - company: 光线传媒
    code: "300251.SZ"
    market: A股
    films:
      - ids: [1462628]
        names: [欢迎来龙餐馆]
        role: producer          # producer | distributor
        core: true              # 对该公司业绩弹性大时标记
        note: 联合出品；依据：…
```

一部片可挂在多家公司下。`core: true` 在总结段优先展示。

## 4. 分析逻辑

与 game-wind **不同**：不做「上升≥N / 新进前50 / 占比跃升 / 日票房金额环比」等通用异动引擎。

### 4.1 核心对象

1. 用日票房给在映片排序，取 **Top10**（并列规则：日票房降序，其次累计，再次 `movieId`，实现时写死并测）
2. 与 **上一份成功快照** 的日票房 Top10 对比名次

### 4.2 名次标记

| 标记 | 含义 |
|------|------|
| `新` | 上次不在 Top10（或无对比基线时不做「新」，见首次运行） |
| `↑N` / `↓N` | 仍在 Top10，名次相对上次变化 N |
| `—` | 名次不变 |

**不关心**：日票房金额增减、占比变化、Top10 以外涨跌。

### 4.3 总结段素材

仅基于 Top10 相对上次：

- 新进前十：片名 + 关联白名单公司
- 名次明显变动：升/降 **≥ N**（配置项，默认 `summary_rank_move: 3`）+ 公司
- 多公司关联：一部片挂 ≥2 家白名单公司时点明
- `core` 片优先
- 若无上述变化：一句「Top10 成员与名次相对稳定」

## 5. 报告结构

Markdown，推企业微信（超长按行拆条；同日同内容指纹防重，对齐 game-wind）。

```markdown
# 电影公司观察 · 2026-08-21
> 数据源：猫眼在映榜 · 日票房=累计差分 · 与最近一次快照对比
> 抓取状态：…（仅失败时）

## 一、日票房 Top10
1. 《片名》(id) 第1(↑2) · 日票房 xx万 · 累计 … · 关联：公司A(出品)、公司B(发行)
…

## 二、影视公司
### 光线传媒 · 300251.SZ
- 《…》日票房第1(↑2) · …
### 某公司 · 代码
- 今日 Top10 无关联影片

## 三、总结
值得关注：
- …

## 四、待补全
- 第3 《空枪》 movieId=…（请核实出品方是否为上市标的）
```

段落约定：

1. **标题**：日期、数据源、抓取状态  
2. **日票房 Top10**：完整 1–10 + 标记 + 关联公司  
3. **影视公司**：按 `movies.yaml` **全部公司**分段；有 Top10 关联则列明细，否则一句「今日 Top10 无关联影片」  
4. **总结**：见 §4.3  
5. **待补全**：Top10 中未映射到白名单的片子 + `movieId`

### 5.1 推送策略

- **每日推送完整报告**（即使 Top10 相对上次全是 `—`、无新进）
- 配置项：`notify.notify_when_quiet`，**默认 `true`**
- 同日相同内容指纹不重复推送

## 6. 架构与命令

技术栈：**Go** CLI（对齐 game-wind）。

```
cron 08:00 → moviewind run
  fetch → store → analyze → report → notify
```

| 命令 | 作用 |
|------|------|
| `moviewind run [--dry-run] [--date YYYY-MM-DD]` | 抓取 → 分析 → 报告 → 推送 |
| `moviewind fetch` | 只抓取入库 |
| `moviewind report [--dry-run] [--date]` | 用已有快照出报告/推送 |
| `moviewind list` | 列出已有快照 |
| `--data-dir` | 默认 `data/` |

包划分（示意）：

- `cmd/moviewind`
- `internal/fetch`
- `internal/store`
- `internal/mapping`
- `internal/analyze`
- `internal/report`
- `internal/notify`
- `internal/config`

配置：

- `config.yaml`（入库）：`top_n: 10`、`summary_rank_move: 3`、`notify.notify_when_quiet: true` 等
- `config.local.yaml`（gitignore）：`notify.webhook_url`

快照：`data/YYYY-MM-DD.json`；状态：`data/state.json`；报告可选落盘 `data/reports/YYYY-MM-DD.md`。

## 7. 已确认决策一览

| 项 | 决定 |
|----|------|
| 产品形态 | 仿 game-wind，服务电影股投资者 |
| 数据主源 | 猫眼 `dashboard-ajax/movie` |
| 日票房 | 累计差分；不解码字体字段 |
| 映射 | yaml 白名单；默认出品；重要发行可选+note |
| 定时 | 外置 cron；建议 08:00；无内置 scheduler |
| 结算与推送 | 默认同一次 `run`；命令可拆 |
| 日期文案 | 运行日历日，不强调昨/今营业日 |
| 分析 | 只盯日票房 Top10 + ↑↓新；不管金额增减 |
| 报告 | Top10 → 全公司 → 总结 → 待补全 |
| 安静日 | 仍推送；`notify_when_quiet` 默认 true |
| 按公司段 | yaml 全部公司 |
| 语言 | Go |
| 仓库 | 已 `git init` |

## 8. 风险与后续

- 猫眼接口可能加签名 / 登录 / 封禁：需失败可见 + 日后备用源
- 累计差分 ≠ 官方日结票房，报告中保持「与上次快照对比」表述
- 二期可选：盘中小时快照、金额类信号、历史回填（如 Tushare）

## 9. 实现顺序（进入计划后展开）

1. 模块骨架 + config + store  
2. fetch 猫眼 + 累计解析  
3. movies.yaml 映射 + 维护提示词  
4. analyze：差分、Top10、标记、总结素材  
5. report + notify  
6. CLI `run/fetch/report/list` + README + 示例 cron  
