#!/usr/bin/env bash
# Bing Search API 调用示例
# 用法：先启动服务 `go run . -addr :8080`，再执行 `bash examples/curl.sh`
set -u

BASE="${BASE:-http://127.0.0.1:8080}"

show() {
  echo
  echo "========== $1 =========="
  shift
  curl -sS --max-time 40 "$@" | python3 -m json.tool 2>/dev/null || echo "(响应不是 JSON 或服务未启动)"
}

show "健康检查" "$BASE/health"

show "网页搜索" "$BASE/search/web?q=golang+tutorial&count=3"

show "网页搜索 - 站点限定 + 时间筛选" \
  "$BASE/search/web?q=docker+compose&site=docs.docker.com&freshness=week&count=3"

show "网页搜索 - 精确匹配 + 排除词 + 分页" \
  "$BASE/search/web?q=go+web+framework&exact_match=true&exclude=gin&page=2&count=3"

show "图片搜索" "$BASE/search/images?q=go+language+logo&count=3"

show "视频搜索" "$BASE/search/videos?q=golang&count=3"

show "资讯搜索" "$BASE/search/news?q=artificial+intelligence&count=3"

show "多频道一次返回" "$BASE/search/all?q=redis&types=web,images&count=2"

show "联想词" "$BASE/suggest?q=gol&market=en-US"

show "POST JSON 请求" \
  -X POST "$BASE/search/web" \
  -H 'Content-Type: application/json' \
  -d '{"q":"http server","count":2,"site":"go.dev","language":"en-US"}'

show "透传 Bing 原生参数" "$BASE/search/web?q=go&b_form=QBRE&b_cc=US&count=2"
