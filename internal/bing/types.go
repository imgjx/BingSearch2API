package bing

// SearchOptions 描述一次搜索请求的所有可配置项。
// 零值表示使用 Bing 的默认值。
type SearchOptions struct {
	Query      string            // 搜索关键词（必填）
	Count      int               // 期望返回条数，1-50，默认 10
	Offset     int               // 结果偏移量（从 0 开始）
	Language   string            // 界面语言，例如 en-US / zh-CN（setlang）
	Region     string            // 地区代码，例如 US / CN（cc）
	Market     string            // 市场，例如 en-US / zh-CN（mkt），优先级高于 Language+Region
	SafeSearch string            // off / moderate / strict
	Freshness  string            // day / week / month / year，或 Bing 原生 ez1/ez2/...
	Site       string            // 限定站点，例如 github.com（site:）
	FileType   string            // 限定文件类型，例如 pdf（filetype:）
	Exclude    string            // 排除关键词，空格分隔（-keyword）
	ExactMatch bool              // 整句精确匹配（加引号）
	Filters    string            // Bing 原生 filters 参数，例如 `ex1:"ez1"`
	QFT        string            // Bing 原生 qft 参数，例如 +filterui:age-lt1440
	Extra      map[string]string // 其它原样透传给 Bing 的查询参数，用于完整自定义
}

// Sitelink 是搜索结果下方的子链接。
type Sitelink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// WebResult 是一条网页搜索结果。
type WebResult struct {
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	DisplayURL    string     `json:"display_url"`
	Description   string     `json:"description"`
	SiteName      string     `json:"site_name,omitempty"`
	PublishedDate string     `json:"published_date,omitempty"`
	Sitelinks     []Sitelink `json:"sitelinks,omitempty"`
}

// ImageResult 是一条图片搜索结果。
type ImageResult struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	URL          string `json:"url"`           // 原图地址 murl
	ThumbnailURL string `json:"thumbnail_url"` // 缩略图地址 turl
	PageURL      string `json:"page_url"`      // 图片所在网页 purl
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	FileSize     string `json:"file_size,omitempty"`
	FileType     string `json:"file_type,omitempty"`
	ID           string `json:"id,omitempty"`
}

// VideoResult 是一条视频搜索结果。
type VideoResult struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Description  string `json:"description,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Source       string `json:"source,omitempty"`
	Published    string `json:"published,omitempty"`
	Uploader     string `json:"uploader,omitempty"`
}

// NewsResult 是一条资讯搜索结果。
type NewsResult struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	Description  string `json:"description,omitempty"`
	Source       string `json:"source,omitempty"`
	Published    string `json:"published,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

// SuggestItem 是一条搜索建议。
type SuggestItem struct {
	Text string `json:"text"`
	Type string `json:"type,omitempty"`
}

// Page 是垂直搜索的通用结果页。
type Page[T any] struct {
	Query          string `json:"query"`
	Results        []T    `json:"results"`
	TotalEstimated string `json:"total_estimated,omitempty"`
	RequestURL     string `json:"request_url,omitempty"`
	Fallback       bool   `json:"fallback,omitempty"` // 上游不可用时的降级标记
	Note           string `json:"note,omitempty"`
}
