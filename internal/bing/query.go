package bing

import (
	"net/url"
	"strconv"
	"strings"
)

const baseURL = "https://www.bing.com"

var verticalPath = map[string]string{
	"web":          "/search",
	"images":       "/images/search",
	"videos":       "/videos/search",
	"videos_async": "/videos/asyncv2",
	"news":         "/news/search",
}

var verticalForm = map[string]string{
	"web":    "QBLH",
	"images": "IRFLTR",
	"videos": "VIDPER",
	"news":   "YFNR",
}

// buildQuery 把结构化选项拼成 Bing 的 q 参数。
func buildQuery(o SearchOptions) string {
	q := strings.TrimSpace(o.Query)
	if o.ExactMatch && !isQuoted(q) {
		q = `"` + q + `"`
	}
	if o.Site != "" {
		q += " site:" + o.Site
	}
	if o.FileType != "" {
		q += " filetype:" + o.FileType
	}
	for _, word := range strings.Fields(o.Exclude) {
		q += " -" + strings.TrimPrefix(word, "-")
	}
	return strings.TrimSpace(q)
}

func isQuoted(s string) bool {
	runes := []rune(s)
	if len(runes) < 2 {
		return false
	}
	first, last := runes[0], runes[len(runes)-1]
	return (first == '\u0022' && last == '\u0022') || (first == '\u201c' && last == '\u201d')
}

// freshnessToFilter 将易读的时间范围映射为 Bing 的 filters 值。
// 若用户已经显式传入 Filters，则以用户传入的为准。
func freshnessToFilter(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "day", "24h", "d":
		return `ex1:"ez1"`
	case "week", "w":
		return `ex1:"ez2"`
	case "month", "m":
		return `ex1:"ez3"`
	case "year", "y":
		return `ex1:"ez5"`
	default:
		// 允许直接传 ez1 / ez2 / ez3 / ez5 等 Bing 原生值
		if strings.HasPrefix(strings.ToLower(f), "ez") {
			return `ex1:"` + f + `"`
		}
		return ""
	}
}

// buildURL 根据选项生成完整的 Bing 请求地址。
func buildURL(vertical string, o SearchOptions) string {
	path, ok := verticalPath[vertical]
	if !ok {
		path = verticalPath["web"]
	}
	u, _ := url.Parse(baseURL + path)
	q := u.Query()

	q.Set("q", buildQuery(o))
	if o.Count > 0 {
		q.Set("count", strconv.Itoa(o.Count))
	}
	if o.Offset > 0 {
		q.Set("first", strconv.Itoa(o.Offset+1))
	}
	if o.Market != "" {
		q.Set("mkt", o.Market)
	} else {
		if o.Language != "" {
			q.Set("setlang", o.Language)
		}
		if o.Region != "" {
			q.Set("cc", o.Region)
		}
	}
	switch strings.ToLower(o.SafeSearch) {
	case "off", "moderate", "strict":
		q.Set("adlt", strings.ToLower(o.SafeSearch))
	}
	if o.Filters != "" {
		q.Set("filters", o.Filters)
	} else if f := freshnessToFilter(o.Freshness); f != "" {
		q.Set("filters", f)
	}
	if o.QFT != "" {
		q.Set("qft", o.QFT)
	}
	if form, ok := verticalForm[vertical]; ok {
		q.Set("form", form)
	}
	for k, v := range o.Extra {
		if k != "" && v != "" {
			q.Set(k, v)
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// buildVideoAsyncURL 生成视频异步加载接口地址。
// Bing 的视频页经常只渲染首条结果，异步接口通常能返回完整列表，
// 因此两者互补使用以提高召回率。
func buildVideoAsyncURL(o SearchOptions) string {
	u, _ := url.Parse(baseURL + verticalPath["videos_async"])
	q := u.Query()
	q.Set("q", buildQuery(o))
	q.Set("async", "content")
	q.Set("mmasync", "1")
	q.Set("vdpp", "VideoResultAsync")
	q.Set("first", strconv.Itoa(o.Offset+1))
	if o.Count > 0 {
		q.Set("count", strconv.Itoa(o.Count))
	}
	if o.Market != "" {
		q.Set("mkt", o.Market)
	}
	if o.Language != "" {
		q.Set("setlang", o.Language)
	}
	if o.Filters != "" {
		q.Set("filters", o.Filters)
	} else if f := freshnessToFilter(o.Freshness); f != "" {
		q.Set("filters", f)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// buildSuggestURL 生成联想词请求地址。
func buildSuggestURL(q, market string) string {
	u, _ := url.Parse(baseURL + "/AS/Suggestions")
	val := u.Query()
	val.Set("pt", "page.serp")
	val.Set("q", q)
	if market == "" {
		market = "en-US"
	}
	val.Set("mkt", market)
	val.Set("cvid", "BINGSEARCHAPI")
	u.RawQuery = val.Encode()
	return u.String()
}
