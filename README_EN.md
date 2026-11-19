# 🔍 Bing Search API

[🇨🇳 中文](README.md) · [🇺🇸 English](README_EN.md)

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)]()
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)]()

> Turn Bing web search into a simple HTTP JSON API — no API key required.

A lightweight Go service that parses Bing search result pages and exposes a unified RESTful API.
Useful for personal projects, internal tools and learning scenarios where you need search results without an official API key.

> ⚠️ This is an **unofficial** implementation. Please comply with Bing's terms of service and keep request rates reasonable. See the [Disclaimer](#-disclaimer).

---

## ✨ Features

- ✅ **Five search verticals**: web, images, videos, news and suggestions, plus a multi-vertical aggregation endpoint
- ✅ **No API key**: no Azure or Bing official credentials needed
- ✅ **Full query control**: pagination, language, region, safe search, freshness, `site:`, `filetype:`, excluded terms, exact match
- ✅ **Native parameter passthrough**: forward any Bing parameter via `b_` prefixed query params or an `extra` object
- ✅ **Production-minded**: timeouts, retries, cookie sessions, per-IP rate limiting, CORS, request IDs and graceful shutdown
- ✅ **Smart fallback**: falls back to web search when the news vertical is unavailable, clearly flagged in the response
- ✅ **Consistent JSON**: the same success/error envelope across all endpoints; error messages in English
- ✅ **Lightweight**: only `golang.org/x/net/html` as a dependency, with unit tests included

## 📑 Table of Contents

- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [API Reference](#-api-reference)
- [Configuration](#-configuration)
- [Project Structure](#-project-structure)
- [Roadmap](#-roadmap)
- [Contributing](#-contributing)
- [License](#-license)
- [Disclaimer](#-disclaimer)

## 📦 Installation

**Requirements**: Go 1.21 or newer.

```bash
git clone https://github.com/imgjx/BingSearch2API.git
cd BingSearch2API
go build -o bing-search-api .
```

Or run directly:

```bash
go run . -addr :8080
```

## 🚀 Quick Start

Start the server:

```bash
./bing-search-api -addr :8080
```

Health check:

```bash
curl http://127.0.0.1:8080/health
# {"service":"bing-search-api","status":"ok"}
```

Your first search:

```bash
curl "http://127.0.0.1:8080/search/web?q=golang+tutorial&count=3"
```

Example response:

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

More runnable examples are in [`examples/curl.sh`](examples/curl.sh):

```bash
bash examples/curl.sh
```

## 📖 API Reference

### Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET / POST | `/search/web` | Web search |
| GET / POST | `/search/images` | Image search |
| GET / POST | `/search/videos` | Video search |
| GET / POST | `/search/news` | News search |
| GET / POST | `/search/all` | Multiple verticals in one request |
| GET / POST | `/suggest` | Search suggestions |
| GET | `/health` | Health check |
| OPTIONS | any | CORS preflight |

### Examples

```bash
# Web search
curl "http://127.0.0.1:8080/search/web?q=golang+tutorial&count=5"

# Site restriction + past week
curl "http://127.0.0.1:8080/search/web?q=docker&site=docs.docker.com&freshness=week"

# Images / videos / news
curl "http://127.0.0.1:8080/search/images?q=go+logo&count=10"
curl "http://127.0.0.1:8080/search/videos?q=golang&count=5"
curl "http://127.0.0.1:8080/search/news?q=AI&count=5"

# Multiple verticals
curl "http://127.0.0.1:8080/search/all?q=redis&types=web,images&count=3"

# Suggestions
curl "http://127.0.0.1:8080/suggest?q=gol&market=en-US"
```

POST with JSON (can be mixed with query parameters; query parameters take precedence):

```bash
curl -X POST http://127.0.0.1:8080/search/web \
  -H 'Content-Type: application/json' \
  -d '{"q":"docker compose","count":5,"freshness":"week","site":"docs.docker.com"}'
```

### Query Parameters

| Parameter | Aliases | Type | Description |
| --- | --- | --- | --- |
| `q` | `query` | string | **Required**, search query |
| `count` | `limit` | int | Number of results, 1-50, default 10 |
| `offset` | - | int | Result offset (0-based) |
| `page` | - | int | Page number (1-based); overrides `offset` when set |
| `language` | `lang`, `setlang` | string | UI language, e.g. `en-US`, `zh-CN` |
| `region` | `cc`, `country` | string | Region, e.g. `US`, `CN` |
| `market` | `mkt` | string | Market, e.g. `en-US`; takes precedence over language + region |
| `safeSearch` | `safe`, `safe_search`, `adlt` | string | `off` / `moderate` / `strict` |
| `freshness` | `time` | string | `day` / `week` / `month` / `year`, or `ez1`/`ez2`/`ez3`/`ez5` |
| `site` | - | string | Restrict to a site, e.g. `github.com` (generates `site:`) |
| `fileType` | `file_type`, `filetype` | string | Restrict file type, e.g. `pdf` (generates `filetype:`) |
| `exclude` | `exclude_words` | string | Excluded terms, space separated (generates `-keyword`) |
| `exactMatch` | `exact`, `exact_match` | bool | Exact phrase match (adds quotes) |
| `filters` | - | string | Native Bing `filters`, e.g. `ex1:"ez2"` |
| `qft` | - | string | Native Bing `qft`, e.g. `+filterui:age-lt1440` |
| `types` | - | string | `/search/all` only: `web,images,videos,news,suggest` |

### Native Bing Parameter Passthrough

```bash
# 1) b_ prefixed query parameters (b_form -> form)
curl "http://127.0.0.1:8080/search/web?q=go&b_form=QBRE&b_cc=US"

# 2) extra object in JSON body
curl -X POST http://127.0.0.1:8080/search/web \
  -H 'Content-Type: application/json' \
  -d '{"q":"go","extra":{"form":"QBRE","cc":"US"}}'
```

### Response Format

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

Error response (all error messages are in English):

```json
{"success": false, "count": 0, "offset": 0, "error": "missing required parameter: q (search query)", "request_id": "..."}
```

Result fields by vertical:

| Vertical | Fields |
| --- | --- |
| web | `title`, `url`, `display_url`, `description`, `site_name`, `published_date`, `sitelinks` |
| images | `title`, `description`, `url` (original), `thumbnail_url`, `page_url`, `width`, `height`, `file_size`, `id` |
| videos | `title`, `url`, `thumbnail_url`, `description`, `duration`, `source`, `published`, `uploader` |
| news | `title`, `url`, `description`, `source`, `published`, `thumbnail_url` |
| suggest | `text`, `type` |

## 🔧 Configuration

Configure via command-line flags:

| Flag | Default | Description |
| --- | --- | --- |
| `-addr` | `:8080` | Listen address |
| `-timeout` | `20s` | Timeout for upstream Bing requests |
| `-rate` | `20` | Allowed requests per second per client IP |

Development checks:

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## 🗂 Project Structure

```
BingSearch2API/
├── main.go                      # Entry point, graceful shutdown
├── go.mod / go.sum
├── LICENSE                      # MIT license
├── README.md                    # 中文文档
├── README_EN.md                 # English docs
├── internal/
│   ├── bing/
│   │   ├── client.go            # HTTP client, retries, timeouts, cookies
│   │   ├── query.go             # Query building and Bing search syntax
│   │   ├── parser.go            # HTML parsing (web/images/videos/news/suggest)
│   │   ├── search.go            # Orchestration, fallback, result trimming
│   │   ├── types.go             # Data structures
│   │   └── bing_test.go         # Parser and query unit tests
│   └── api/
│       ├── server.go            # Routes, middleware, rate limiting
│       └── handlers.go          # Parameter parsing and response envelope
└── examples/curl.sh             # Runnable usage examples
```

## 🛣 Roadmap

- [ ] Add a Docker image and `docker-compose` example
- [ ] Add concurrent fetching and a cache layer to reduce upstream load
- [ ] Add GitHub Actions for automated tests and builds
- [ ] Wrap more native Bing filter parameters
- [ ] Provide an OpenAPI / Swagger specification

## 🤝 Contributing

Issues and pull requests are welcome!

1. Fork this repository
2. Create a branch: `git checkout -b feature/your-feature`
3. Run `go test ./...` and `gofmt -w .` before committing
4. Push your branch and open a pull request

When filing an issue, please include: the request URL (redact sensitive values), expected result, actual result and your environment.

## 📄 License

Released under the [MIT License](LICENSE).

## ❗ Disclaimer

1. This is an **unofficial** wrapper around Bing search and is not affiliated with Microsoft.
2. Results depend on Bing's region, language, session and anti-bot policies. Comply with Bing's terms and control request rates in production.
3. Known limitations:
   - Some regions (e.g. `cn.bing.com`) redirect the news vertical. In that case `/search/news` automatically falls back to web search and the response contains `fallback: true`.
   - Bing's suggestions endpoint returns 204 on some datacenter IPs; `/suggest` returns an empty array with a `note`.
   - Video result counts may vary by Bing session. The service merges the main page with an async endpoint, but may still return fewer than `count`.
4. Bing page changes may break parsing; update `internal/bing/parser.go` when that happens.
