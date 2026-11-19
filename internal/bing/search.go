package bing

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Web 执行网页搜索。
func (c *Client) Web(ctx context.Context, opts SearchOptions) (*Page[WebResult], error) {
	u := buildURL("web", opts)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	page := parseWeb(body)
	trimResults(&page.Results, opts.Count)
	page.Query = opts.Query
	page.RequestURL = u
	return page, nil
}

// Images 执行图片搜索。
func (c *Client) Images(ctx context.Context, opts SearchOptions) (*Page[ImageResult], error) {
	u := buildURL("images", opts)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	page := parseImages(body)
	trimResults(&page.Results, opts.Count)
	page.Query = opts.Query
	page.RequestURL = u
	return page, nil
}

// Videos 执行视频搜索。
// 由于 Bing 视频页在不同会话下可能只渲染少量结果，这里同时请求主页面与
// 异步加载接口，按 URL 去重后合并，尽量达到用户请求的条数。
func (c *Client) Videos(ctx context.Context, opts SearchOptions) (*Page[VideoResult], error) {
	u := buildURL("videos", opts)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	page := parseVideos(body)
	page.Query = opts.Query
	page.RequestURL = u

	if len(page.Results) >= opts.Count {
		trimResults(&page.Results, opts.Count)
		return page, nil
	}

	asyncURL := buildVideoAsyncURL(opts)
	asyncBody, asyncErr := c.get(ctx, asyncURL)
	if asyncErr == nil && len(asyncBody) > 0 {
		asyncPage := parseVideos(asyncBody)
		seen := map[string]bool{}
		for _, r := range page.Results {
			seen[r.URL] = true
		}
		for _, r := range asyncPage.Results {
			if seen[r.URL] {
				continue
			}
			seen[r.URL] = true
			page.Results = append(page.Results, r)
		}
	}
	trimResults(&page.Results, opts.Count)
	return page, nil
}

// News 执行资讯搜索。
// 某些地区（例如 cn.bing.com）会屏蔽 news 垂直频道并重定向到首页，
// 此时自动降级为「网页搜索 + 时间筛选」，并把结果映射为资讯结构。
func (c *Client) News(ctx context.Context, opts SearchOptions) (*Page[NewsResult], error) {
	u := buildURL("news", opts)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	page := parseNews(body)
	page.Query = opts.Query
	page.RequestURL = u

	if len(page.Results) > 0 {
		return page, nil
	}

	// 降级：用网页搜索结果构造资讯结果
	fallbackOpts := opts
	if fallbackOpts.Filters == "" && fallbackOpts.Freshness == "" {
		fallbackOpts.Filters = `ex1:"ez2"` // 最近一周
	}
	webPage, err := c.Web(ctx, fallbackOpts)
	if err != nil {
		page.Note = "Bing News is unavailable and the web-search fallback also failed: " + err.Error()
		return page, nil
	}
	page.Fallback = true
	page.Note = "Bing News is not accessible from the current region; automatically using web search results with a time filter instead"
	page.TotalEstimated = webPage.TotalEstimated
	page.RequestURL = webPage.RequestURL
	for _, w := range webPage.Results {
		page.Results = append(page.Results, NewsResult{
			Title:       w.Title,
			URL:         w.URL,
			Description: w.Description,
			Source:      w.SiteName,
			Published:   w.PublishedDate,
		})
	}
	trimResults(&page.Results, opts.Count)
	return page, nil
}

// trimResults 将结果截断到用户请求的条数（Bing 常返回固定 10 条或更多）。
func trimResults[T any](results *[]T, count int) {
	if count > 0 && len(*results) > count {
		*results = (*results)[:count]
	}
}

// Suggest 获取搜索联想词。
func (c *Client) Suggest(ctx context.Context, q, market string) ([]SuggestItem, error) {
	if strings.TrimSpace(q) == "" {
		return []SuggestItem{}, nil
	}
	u := buildSuggestURL(q, market)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	items := parseSuggestions(body)
	if len(items) == 0 {
		// 部分网络环境下 Bing 的联想词接口返回 204；此处给出基于网页搜索的兜底建议。
		items = c.suggestFallback(ctx, q, market)
	}
	return items, nil
}

// suggestFallback 从搜索结果页的「相关搜索」区域提取建议词。
func (c *Client) suggestFallback(ctx context.Context, q, market string) []SuggestItem {
	opts := SearchOptions{Query: q, Count: 10, Market: market}
	u := buildURL("web", opts)
	body, err := c.get(ctx, u)
	if err != nil {
		return []SuggestItem{}
	}
	doc, err := parseDocument(body)
	if err != nil {
		return []SuggestItem{}
	}
	seen := map[string]bool{}
	out := []SuggestItem{}
	for _, a := range descendants(doc, "a", "") {
		cls := getAttr(a, "class")
		if !strings.Contains(cls, "b_rs") && !strings.Contains(cls, "related") && !strings.Contains(cls, "sugg") {
			continue
		}
		t := textOf(a)
		if t == "" || len(t) > 60 || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, SuggestItem{Text: t, Type: "related"})
		if len(out) >= 10 {
			break
		}
	}
	return out
}

// All 一次请求同时返回多个垂直频道的结果。
func (c *Client) All(ctx context.Context, opts SearchOptions, verticals []string) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	var errs []string
	for _, v := range verticals {
		switch v {
		case "web":
			p, err := c.Web(ctx, opts)
			if err != nil {
				errs = append(errs, "web: "+err.Error())
			} else {
				out[v] = p
			}
		case "images":
			p, err := c.Images(ctx, opts)
			if err != nil {
				errs = append(errs, "images: "+err.Error())
			} else {
				out[v] = p
			}
		case "videos":
			p, err := c.Videos(ctx, opts)
			if err != nil {
				errs = append(errs, "videos: "+err.Error())
			} else {
				out[v] = p
			}
		case "news":
			p, err := c.News(ctx, opts)
			if err != nil {
				errs = append(errs, "news: "+err.Error())
			} else {
				out[v] = p
			}
		case "suggest":
			p, err := c.Suggest(ctx, opts.Query, opts.Market)
			if err != nil {
				errs = append(errs, "suggest: "+err.Error())
			} else {
				out[v] = p
			}
		default:
			errs = append(errs, fmt.Sprintf("unknown vertical: %s", v))
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		out["_errors"] = errs
	}
	return out, nil
}

// Validate 对选项做基本校验与默认值填充。
func (o *SearchOptions) Validate() error {
	o.Query = strings.TrimSpace(o.Query)
	if o.Query == "" {
		return fmt.Errorf("missing required parameter: q (search query)")
	}
	if o.Count <= 0 {
		o.Count = 10
	}
	if o.Count > 50 {
		o.Count = 50
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	if o.Market == "" && o.Language == "" && o.Region == "" {
		o.Language = "en-US"
	}
	if o.Extra == nil {
		o.Extra = map[string]string{}
	}
	if _, err := url.Parse(o.Site); err != nil && o.Site != "" {
		return fmt.Errorf("invalid site parameter: %w", err)
	}
	return nil
}
