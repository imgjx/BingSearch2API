package bing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// ---------- 通用 DOM 辅助函数 ----------

func parseDocument(body []byte) (*html.Node, error) {
	return html.Parse(bytes.NewReader(body))
}

func hasClass(n *html.Node, name string) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == name {
					return true
				}
			}
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	if n == nil || n.Type != html.ElementNode {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

// descendants 返回所有满足 tag（空表示任意）与 class（空表示任意）的后代节点，含自身。
func descendants(n *html.Node, tag, class string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.ElementNode &&
			(tag == "" || strings.EqualFold(x.Data, tag)) &&
			(class == "" || hasClass(x, class)) {
			out = append(out, x)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	if n != nil {
		walk(n)
	}
	return out
}

func descendant(n *html.Node, tag, class string) *html.Node {
	list := descendants(n, tag, class)
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

func textOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

var resultCountRe = regexp.MustCompile(`[\d][\d,\.]*`)

func extractTotal(doc *html.Node) string {
	for _, cls := range []string{"sb_count", "b_pageCount"} {
		for _, n := range descendants(doc, "", cls) {
			if m := resultCountRe.FindString(textOf(n)); m != "" {
				return m
			}
		}
	}
	return ""
}

// ---------- 网页搜索 ----------

func parseWeb(body []byte) *Page[WebResult] {
	page := &Page[WebResult]{Results: []WebResult{}}
	doc, err := parseDocument(body)
	if err != nil {
		return page
	}
	page.TotalEstimated = extractTotal(doc)

	for _, li := range descendants(doc, "li", "b_algo") {
		var r WebResult
		if h2 := descendant(li, "h2", ""); h2 != nil {
			if a := descendant(h2, "a", ""); a != nil {
				r.Title = textOf(a)
				r.URL = normalizeURL(getAttr(a, "href"))
			}
		}
		if cap := descendant(li, "div", "b_caption"); cap != nil {
			if p := descendant(cap, "p", ""); p != nil {
				r.Description = textOf(p)
			}
		}
		if cite := descendant(li, "cite", ""); cite != nil {
			r.DisplayURL = textOf(cite)
		}
		if t := descendant(li, "div", "tptt"); t != nil {
			r.SiteName = textOf(t)
		}
		r.PublishedDate = extractPublished(li)

		seen := map[string]bool{r.URL: true}
		for _, a := range descendants(li, "a", "") {
			href := normalizeURL(getAttr(a, "href"))
			title := textOf(a)
			if href == "" || r.URL == "" || href == r.URL || seen[href] || title == "" || len(title) > 80 {
				continue
			}
			// 子链接必须属于同一个域名，避免把"相关内容"等推荐链接混入。
			if hostOf(href) != hostOf(r.URL) {
				continue
			}
			seen[href] = true
			r.Sitelinks = append(r.Sitelinks, Sitelink{Title: title, URL: href})
			if len(r.Sitelinks) >= 8 {
				break
			}
		}

		if r.Title != "" && r.URL != "" {
			page.Results = append(page.Results, r)
		}
	}
	return page
}

func extractPublished(n *html.Node) string {
	for _, cls := range []string{"news_dt", "b_date", "b_factrow"} {
		for _, x := range descendants(n, "span", cls) {
			t := textOf(x)
			if t != "" && len(t) < 60 {
				return t
			}
		}
	}
	return ""
}

// ---------- 图片搜索 ----------

type imageMeta struct {
	T    string      `json:"t"`
	MURL string      `json:"murl"`
	TURL string      `json:"turl"`
	PURL string      `json:"purl"`
	Desc string      `json:"desc"`
	MID  string      `json:"mid"`
	FS   string      `json:"fs"`
	W    interface{} `json:"w"`
	H    interface{} `json:"h"`
}

func toInt(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(x))
		return i
	default:
		return 0
	}
}

func parseImages(body []byte) *Page[ImageResult] {
	page := &Page[ImageResult]{Results: []ImageResult{}}
	doc, err := parseDocument(body)
	if err != nil {
		return page
	}
	seen := map[string]bool{}
	for _, a := range descendants(doc, "a", "iusc") {
		raw := getAttr(a, "m")
		if raw == "" {
			continue
		}
		var m imageMeta
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			continue
		}
		if m.MURL == "" || seen[m.MURL] {
			continue
		}
		seen[m.MURL] = true
		page.Results = append(page.Results, ImageResult{
			Title:        m.T,
			Description:  m.Desc,
			URL:          m.MURL,
			ThumbnailURL: m.TURL,
			PageURL:      m.PURL,
			Width:        toInt(m.W),
			Height:       toInt(m.H),
			FileSize:     m.FS,
			ID:           m.MID,
		})
	}
	return page
}

// ---------- 视频搜索 ----------

func parseVideos(body []byte) *Page[VideoResult] {
	page := &Page[VideoResult]{Results: []VideoResult{}}
	doc, err := parseDocument(body)
	if err != nil {
		return page
	}
	seen := map[string]bool{}
	for _, a := range descendants(doc, "a", "mc_vtvc_link") {
		href := normalizeURL(getAttr(a, "href"))
		if href == "" || seen[href] {
			continue
		}
		seen[href] = true
		r := VideoResult{URL: href}

		if title := descendant(a, "div", "mc_vtvc_title"); title != nil {
			r.Title = getAttr(title, "title")
			if r.Title == "" {
				r.Title = textOf(title)
			}
		}
		if r.Title == "" {
			r.Title = getAttr(a, "aria-label")
		}
		if img := descendant(a, "img", ""); img != nil {
			r.ThumbnailURL = getAttr(img, "data-src-hq")
			if r.ThumbnailURL == "" {
				r.ThumbnailURL = getAttr(img, "data-src")
			}
			if r.ThumbnailURL == "" {
				r.ThumbnailURL = getAttr(img, "src")
			}
		}
		for _, d := range descendants(a, "div", "items") {
			t := textOf(d)
			if strings.Contains(t, ":") && len(t) <= 12 {
				r.Duration = t
				break
			}
		}
		r.Source, r.Published, r.Uploader = parseVideoAria(getAttr(a, "aria-label"))

		if r.Title != "" {
			page.Results = append(page.Results, r)
		}
	}
	return page
}

func parseVideoAria(s string) (source, published, uploader string) {
	for _, seg := range strings.Split(s, "·") {
		seg = strings.TrimSpace(seg)
		switch {
		case strings.HasPrefix(seg, "from "):
			source = strings.TrimSpace(strings.TrimPrefix(seg, "from "))
		case strings.HasPrefix(seg, "uploaded on "):
			published = strings.TrimSpace(strings.TrimPrefix(seg, "uploaded on "))
		case strings.HasPrefix(seg, "uploaded by "):
			uploader = strings.TrimSpace(strings.TrimPrefix(seg, "uploaded by "))
		}
	}
	return
}

// ---------- 资讯搜索 ----------

func parseNews(body []byte) *Page[NewsResult] {
	page := &Page[NewsResult]{Results: []NewsResult{}}
	doc, err := parseDocument(body)
	if err != nil {
		return page
	}
	page.TotalEstimated = extractTotal(doc)

	selectors := []struct{ tag, class string }{
		{"div", "news-card"}, {"li", "news-card"}, {"div", "newsitem"},
		{"div", "b_algo"}, {"li", "b_algo"},
	}
	seen := map[string]bool{}
	for _, sel := range selectors {
		for _, card := range descendants(doc, sel.tag, sel.class) {
			var r NewsResult
			link := descendant(card, "a", "title")
			if link == nil {
				link = descendant(card, "a", "tilk")
			}
			if link == nil {
				if h := descendant(card, "h2", ""); h != nil {
					link = descendant(h, "a", "")
				}
			}
			if link == nil {
				link = descendant(card, "a", "")
			}
			if link == nil {
				continue
			}
			r.URL = normalizeURL(getAttr(link, "href"))
			r.Title = textOf(link)
			if r.Title == "" {
				r.Title = getAttr(link, "title")
			}
			if r.URL == "" || r.Title == "" || seen[r.URL] {
				continue
			}
			seen[r.URL] = true

			for _, cls := range []string{"snippet", "b_caption"} {
				if n := descendant(card, "div", cls); n != nil {
					if p := descendant(n, "p", ""); p != nil {
						r.Description = textOf(p)
					}
					if r.Description == "" {
						r.Description = textOf(n)
					}
					if r.Description != "" {
						break
					}
				}
			}
			for _, cls := range []string{"source", "tptt"} {
				if n := descendant(card, "span", cls); n != nil {
					r.Source = textOf(n)
					if r.Source != "" {
						break
					}
				}
			}
			for _, cls := range []string{"time", "news_dt"} {
				if n := descendant(card, "span", cls); n != nil {
					r.Published = textOf(n)
					if r.Published != "" {
						break
					}
				}
			}
			if img := descendant(card, "img", ""); img != nil {
				r.ThumbnailURL = getAttr(img, "data-src")
				if r.ThumbnailURL == "" {
					r.ThumbnailURL = getAttr(img, "src")
				}
			}
			page.Results = append(page.Results, r)
		}
		if len(page.Results) > 0 {
			break
		}
	}
	return page
}

// ---------- 联想词 ----------

type suggestJSON struct {
	Txt  string `json:"txt"`
	Type string `json:"type"`
}

func parseSuggestions(body []byte) []SuggestItem {
	if len(bytes.TrimSpace(body)) == 0 {
		return []SuggestItem{}
	}
	// 优先按 JSON 数组解析
	var js []suggestJSON
	if err := json.Unmarshal(body, &js); err == nil {
		out := make([]SuggestItem, 0, len(js))
		for _, s := range js {
			if s.Txt != "" {
				out = append(out, SuggestItem{Text: s.Txt, Type: s.Type})
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// 兼容 XML / 混合格式：抓取 <Suggestion ...> 节点与 txt 字段
	text := string(body)
	out := []SuggestItem{}
	re := regexp.MustCompile(`(?is)<Suggestion\b[^>]*>(.*?)</Suggestion>`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		inner := m[1]
		var t, typ string
		if tm := regexp.MustCompile(`(?is)<Text>(.*?)</Text>`).FindStringSubmatch(inner); len(tm) > 1 {
			t = tm[1]
		} else if tm := regexp.MustCompile(`(?i)txt\s*[:=]\s*["']([^"']+)`).FindStringSubmatch(inner); len(tm) > 1 {
			t = tm[1]
		}
		if tm := regexp.MustCompile(`(?i)type\s*=\s*["']([^"']+)`).FindStringSubmatch(m[0]); len(tm) > 1 {
			typ = tm[1]
		}
		t = html.UnescapeString(strings.TrimSpace(t))
		if t != "" {
			out = append(out, SuggestItem{Text: t, Type: typ})
		}
	}
	if len(out) > 0 {
		return out
	}
	// 兜底：直接匹配常见的 txt 字段
	re2 := regexp.MustCompile(`"txt"\s*:\s*"([^"]+)"`)
	for _, m := range re2.FindAllStringSubmatch(text, -1) {
		out = append(out, SuggestItem{Text: html.UnescapeString(m[1])})
	}
	return out
}

// ---------- 小工具 ----------

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Bing 的跳转链接 /ck/a?u=a1... 需要解码，普通情况直接返回。
	if strings.HasPrefix(raw, "/ck/a?") {
		u, err := url.Parse(raw)
		if err == nil {
			if v := u.Query().Get("u"); strings.HasPrefix(v, "a1") {
				decoded, derr := base64URLDecode(v[2:])
				if derr == nil {
					return decoded
				}
			}
		}
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return baseURL + raw
	}
	return raw
}

func base64URLDecode(s string) (string, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 {
		s += "="
	}
	b, err := base64.StdEncoding.DecodeString(s)
	return string(b), err
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}
