# 🔍 BingSearch2API

[🇨🇳 中文](README.md) · [🇺🇸 English](README_EN.md)

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)]()
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)]()

> 把 Bing 在线搜索变成一个简单的 HTTP JSON API —— 无需 API Key，开箱即用。

一个用 Go 编写的轻量级服务：直接解析 Bing 搜索结果页，对外提供统一的 RESTful 接口。
适合需要快速接入搜索能力、但不想申请官方 API 密钥的个人项目、内部工具和学习场景。

> ⚠️ 本项目为**非官方**实现，请遵守 Bing 服务条款并合理控制请求频率。详见 [免责声明](#-免责声明)。

---

## ✨ 特性

- ✅ **五种搜索频道**：网页、图片、视频、资讯、联想词，另支持一次请求聚合多个频道
- ✅ **零依赖密钥**：不需要 Azure / Bing 官方 API Key
- ✅ **完整查询参数**：分页、语言、地区、安全搜索、时间筛选、`site:`、`filetype:`、排除词、精确匹配
- ✅ **原生参数透传**：`b_` 前缀或 `extra` 对象可传递任意 Bing 原生参数
- ✅ **生产向设计**：超时重试、Cookie 会话、IP 限流、CORS、request_id、优雅关闭
- ✅ **智能降级**：资讯频道不可达时自动回退到网页搜索，响应中明确标记
- ✅ **统一 JSON 结构**：所有接口使用相同的成功 / 错误响应格式；错误信息为英文
- ✅ **轻量**：仅依赖 `golang.org/x/net/html`，附单元测试

## 📑 目录

- [安装](#-安装)
- [快速开始](#-快速开始)
- [API 文档](#-api-文档)
- [配置](#-配置)
- [项目结构](#-项目结构)
- [路线图](#-路线图)
- [贡献](#-贡献)
- [许可证](#-许可证)
- [免责声明](#-免责声明)

## 📦 安装

**环境要求**：Go 1.21 或更高版本。

```bash
git clone https://github.com/imgjx/BingSearch2API.git
cd BingSearch2API
go build -o bing-search-api .
```

也可以直接运行：

```bash
go run . -addr :8080
```

## 📥 预编译产物

[v0.1.0 Release](https://github.com/imgjx/BingSearch2API/releases/tag/v0.1.0) 提供三端开箱即用包：

| 平台 | 架构 | 文件 |
| --- | --- | --- |
| Windows | x86_64 | `bingsearch2api_0.1.0_windows_amd64.zip` |
| macOS | Intel | `bingsearch2api_0.1.0_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `bingsearch2api_0.1.0_darwin_arm64.tar.gz` |
| Linux | x86_64 | `bingsearch2api_0.1.0_linux_amd64.tar.gz` |

解压后直接运行，例如 Linux / macOS：

```bash
tar -xzf bingsearch2api_0.1.0_linux_amd64.tar.gz
./bingsearch2api -addr :8080
```

Windows（PowerShell）：

```powershell
Expand-Archive bingsearch2api_0.1.0_windows_amd64.zip
.\bingsearch2api.exe -addr :8080
```

## 🚀 快速开始

启动服务：

```bash
./bing-search-api -addr :8080
```

健康检查：

```bash
curl http://127.0.0.1:8080/health
# {"service":"bing-search-api","status":"ok"}
```

发起第一次搜索：

```bash
curl "http://127.0.0.1:8080/search/web?q=golang+tutorial&count=3"
```

返回示例：

```json
{
  "success": true,
  "type": "web",
  "query": "golang tutorial",
  "count": 3,
  "offset": 0,
  "total_estimated": "57,800",
  "results": [
    {
      "title": "The Go Programming Language",
      "url": "https://go.dev/",
      "display_url": "https://go.dev",
      "description": "Get Started Playground Tour...",
      "site_name": "go.dev"
    }
  ],
  "request_url": "https://www.bing.com/search?...",
  "request_id": "3e44ecc5c48583c5"
}
```

更多可直接运行的示例见 [`examples/curl.sh`](examples/curl.sh)：

```bash
bash examples/curl.sh
```

## 📖 API 文档

### 接口列表

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET / POST | `/search/web` | 网页搜索 |
| GET / POST | `/search/images` | 图片搜索 |
| GET / POST | `/search/videos` | 视频搜索 |
| GET / POST | `/search/news` | 资讯搜索 |
| GET / POST | `/search/all` | 一次请求多个频道 |
| GET / POST | `/suggest` | 搜索联想词 |
| GET | `/health` | 健康检查 |
| OPTIONS | 任意 | CORS 预检 |

### 调用示例

```bash
# 网页搜索
curl "http://127.0.0.1:8080/search/web?q=golang+tutorial&count=5"

# 站点限定 + 最近一周
curl "http://127.0.0.1:8080/search/web?q=docker&site=docs.docker.com&freshness=week"

# 图片 / 视频 / 资讯
curl "http://127.0.0.1:8080/search/images?q=go+logo&count=10"
curl "http://127.0.0.1:8080/search/videos?q=golang&count=5"
curl "http://127.0.0.1:8080/search/news?q=AI&count=5"

# 多频道聚合
curl "http://127.0.0.1:8080/search/all?q=redis&types=web,images&count=3"

# 联想词
curl "http://127.0.0.1:8080/suggest?q=gol&market=en-US"
```

POST JSON（与查询参数可混用，查询参数优先）：

```bash
curl -X POST http://127.0.0.1:8080/search/web \
  -H 'Content-Type: application/json' \
  -d '{"q":"docker compose","count":5,"freshness":"week","site":"docs.docker.com"}'
```

### 查询参数

| 参数 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `q` | `query` | string | **必填**，搜索关键词 |
| `count` | `limit` | int | 返回条数 1-50，默认 10 |
| `offset` | - | int | 结果偏移（从 0 开始） |
| `page` | - | int | 页码，从 1 开始；设置后覆盖 `offset` |
| `language` | `lang`, `setlang` | string | 界面语言，如 `en-US`、`zh-CN` |
| `region` | `cc`, `country` | string | 地区，如 `US`、`CN` |
| `market` | `mkt` | string | 市场，如 `en-US`；优先级高于 language + region |
| `safeSearch` | `safe`, `safe_search`, `adlt` | string | `off` / `moderate` / `strict` |
| `freshness` | `time` | string | `day` / `week` / `month` / `year`，或 `ez1`/`ez2`/`ez3`/`ez5` |
| `site` | - | string | 限定站点，如 `github.com`（生成 `site:`） |
| `fileType` | `file_type`, `filetype` | string | 限定文件类型，如 `pdf`（生成 `filetype:`） |
| `exclude` | `exclude_words` | string | 排除词，空格分隔（生成 `-keyword`） |
| `exactMatch` | `exact`, `exact_match` | bool | 整句精确匹配（加引号） |
| `filters` | - | string | Bing 原生 `filters` 参数，例如 `ex1:"ez2"` |
| `qft` | - | string | Bing 原生 `qft` 参数，例如 `+filterui:age-lt1440` |
| `types` | - | string | 仅 `/search/all`：`web,images,videos,news,suggest` |

### 透传任意 Bing 原生参数

```bash
# 方式 1：b_ 前缀（b_form -> form）
curl "http://127.0.0.1:8080/search/web?q=go&b_form=QBRE&b_cc=US"

# 方式 2：JSON 中的 extra 对象
curl -X POST http://127.0.0.1:8080/search/web \
  -H 'Content-Type: application/json' \
  -d '{"q":"go","extra":{"form":"QBRE","cc":"US"}}'
```

### 响应格式

```json
{
  "success": true,
  "type": "web",
  "query": "golang tutorial",
  "count": 2,
  "offset": 0,
  "total_estimated": "57,800",
  "results": [ ... ],
  "request_url": "https://www.bing.com/search?...",
  "request_id": "3e44ecc5c48583c5"
}
```

失败响应（错误信息为英文）：

```json
{"success": false, "count": 0, "offset": 0, "error": "missing required parameter: q (search query)", "request_id": "..."}
```

各频道 `results` 元素字段：

| 频道 | 字段 |
| --- | --- |
| web | `title`、`url`、`display_url`、`description`、`site_name`、`published_date`、`sitelinks` |
| images | `title`、`description`、`url`（原图）、`thumbnail_url`、`page_url`、`width`、`height`、`file_size`、`id` |
| videos | `title`、`url`、`thumbnail_url`、`description`、`duration`、`source`、`published`、`uploader` |
| news | `title`、`url`、`description`、`source`、`published`、`thumbnail_url` |
| suggest | `text`、`type` |

## 🔧 配置

启动时通过命令行参数配置：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-addr` | `:8080` | 监听地址 |
| `-timeout` | `20s` | 请求 Bing 的超时时间 |
| `-rate` | `20` | 每个客户端 IP 每秒允许的请求数 |

开发校验：

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## 🗂 项目结构

```
BingSearch2API/
├── main.go                      # 服务入口、优雅关闭
├── go.mod / go.sum
├── LICENSE                      # MIT 许可证
├── README.md                    # 中文文档
├── README_EN.md                 # English docs
├── internal/
│   ├── bing/
│   │   ├── client.go            # HTTP 客户端、重试、超时、Cookie
│   │   ├── query.go             # 参数拼装与 Bing 查询语法
│   │   ├── parser.go            # HTML 解析（web/images/videos/news/suggest）
│   │   ├── search.go            # 业务编排、降级、结果截断
│   │   ├── types.go             # 数据结构
│   │   └── bing_test.go         # 解析与参数单测
│   └── api/
│       ├── server.go            # 路由、中间件、限流
│       └── handlers.go          # 参数解析与响应封装
└── examples/curl.sh             # 可运行的调用示例
```

## 🛣 路线图

- [ ] 增加 Docker 镜像与 `docker-compose` 示例
- [ ] 增加并发抓取与缓存层，降低上游压力
- [ ] 补充 GitHub Actions 自动测试与构建
- [ ] 完善更多 Bing 原生筛选参数的封装
- [ ] 提供 OpenAPI / Swagger 描述文件

## 🤝 贡献

欢迎 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建分支：`git checkout -b feature/your-feature`
3. 提交改动前运行 `go test ./...` 和 `gofmt -w .`
4. Push 分支并创建 Pull Request

提 Issue 时请附上：请求 URL（可隐去敏感参数）、期望结果、实际结果和运行环境。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源，可自由使用、修改和分发。

## ❗ 免责声明

1. 本项目是**非官方**的 Bing 搜索封装，与 Microsoft 无任何关联。
2. 结果受 Bing 地区、语言、会话和反爬策略影响；生产使用请遵守 Bing 服务条款并控制频率。
3. 已知限制：
   - `cn.bing.com` 等地区会重定向资讯频道，此时 `/search/news` 自动降级为网页搜索，响应含 `fallback: true`。
   - Bing 联想词接口在部分数据中心 IP 上返回 204，`/suggest` 会返回空数组并附 `note`。
   - 视频结果数量受 Bing 会话影响可能波动；服务已合并主页面与异步接口，但仍可能少于 `count`。
4. Bing 页面改版可能导致解析失效，届时需要更新 `internal/bing/parser.go`。
