# 更新 data/movies.yaml —— 给大模型的指令与规范

你是 movie-wind 影视公司观察工具的数据维护员。把本文件连同 `data/movies.yaml` 一并交给你，根据最新猫眼快照与报告「待补全」更新映射表。

## 一、先读

1. `data/movies.yaml` —— 当前映射表  
2. **最新快照** —— `data/` 下日期最新的 `YYYY-MM-DD.json`  
   - **已有快照就直接用**（优先读最新日期文件；也可参考 `data/reports/` 最新报告的「待补全」）  
   - **仅当不存在快照**，或日期明显过期（非最近 1～2 天），且环境可联网时，再执行：  
     ```bash
     source ~/.g/env && go run ./cmd/moviewind fetch
     ```  
3. 需要时联网核实出品 / 联合出品 / 发行方  

## 二、快照结构

```json
{
  "date": "2026-08-21",
  "films": [
    {
      "movie_id": 1462628,
      "name": "欢迎来龙餐馆",
      "sum_box_wan": 129700,
      "box_rate": "31.5%"
    }
  ]
}
```

- `movie_id`：猫眼 ID，匹配主键  
- `name`：片名  

## 三、更新步骤

1. 读最新快照与报告「待补全」中的 Top10 未映射片  
2. 联网核实是否属于 A 股 / 港股通影视公司  
3. 按规范写入 `movies.yaml`  
4. 跑校验（见六）  
5. 汇报改动清单与依据  

## 四、更新规范

1. **只收录 A 股上市公司 + 港股通标的港股。**  
2. 默认角色 `role: producer`（出品 / 联合出品）。  
3. **`distributor` 仅当重要发行**：主发行且对业绩/股价叙事重要，必须写 `note` 依据。普通联合发行、院线、纯宣发代理不收录。  
4. 字段：`ids`（movieId）、`names`（第一个为规范名）、`role`、`core`、`note`。  
5. 匹配优先级：`ids` > `names` 精确。  
6. 一部片可挂多家公司（各写一条）。  
7. 不轻易删除已有条目。  

## 五、归属判定

- 以官方官宣、招股书、主流财经报道为准  
- 联合出品多方都可收录（注明联合出品）  
- 未上市片方不入库  

## 六、校验

```bash
source ~/.g/env && gofmt -w cmd internal && go build ./... && go vet ./... && golangci-lint run ./... && go test ./...
```

期望：lint **0 issues**、测试全部 `ok`。
