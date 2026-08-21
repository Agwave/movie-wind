package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func usage() {
	fmt.Fprintln(os.Stderr, `moviewind — A股/港股影视公司观察工具

用法:
  moviewind run                 抓取 → 分析 → 报告 → 推送
  moviewind fetch               只抓取入库
  moviewind report              用已有快照生成报告（可推送）
  moviewind list                列出已有快照

选项:
  --dry-run                    不推送企业微信
  --date YYYY-MM-DD            指定日期（run/report）
  --data-dir DIR               数据目录（默认 data/）
  --movies FILE                映射表路径（默认 data/movies.yaml）`)
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s（后续任务接线）\n", args[0])
		usage()
		return 2
	}
}
