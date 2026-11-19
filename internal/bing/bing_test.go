package bing

import (
	"strings"
	"testing"
)

const webFixture = `<html><body>
<ul id="b_results">
  <li class="b_algo">
    <div class="b_tpcn"><a class="tilk"><div class="tptt">go.dev</div></a></div>
    <h2><a href="https://go.dev/doc/">The Go Programming Language</a></h2>
    <div class="b_caption"><p>Docs, tutorials and packages for Go.</p></div>
    <cite>https://go.dev › doc</cite>
  </li>
  <li class="b_algo">
    <h2><a href="https://example.com/go">Go 教程</a></h2>
    <div class="b_caption"><p>入门教程。</p></div>
  </li>
</ul>
<div class="sb_count">About 1,234 results</div>
</body></html>`

func TestParseWeb(t *testing.T) {
	page := parseWeb([]byte(webFixture))
	if len(page.Results) != 2 {
		t.Fatalf("期望 2 条结果，实际 %d", len(page.Results))
	}
	first := page.Results[0]
	if first.Title != "The Go Programming Language" {
		t.Errorf("标题错误: %q", first.Title)
	}
	if first.URL != "https://go.dev/doc/" {
		t.Errorf("URL 错误: %q", first.URL)
	}
	if first.Description != "Docs, tutorials and packages for Go." {
		t.Errorf("摘要错误: %q", first.Description)
	}
	if first.SiteName != "go.dev" {
		t.Errorf("站点名错误: %q", first.SiteName)
	}
	if page.TotalEstimated != "1,234" {
		t.Errorf("总数错误: %q", page.TotalEstimated)
	}
}

func TestParseImages(t *testing.T) {
	body := `<a class="iusc" m='{"t":"A Go Logo","murl":"https://img.test/a.png","turl":"https://ts.test/thumb.jpg","purl":"https://page.test","w":800,"h":600}'></a>`
	page := parseImages([]byte(body))
	if len(page.Results) != 1 {
		t.Fatalf("期望 1 张图片，实际 %d", len(page.Results))
	}
	r := page.Results[0]
	if r.URL != "https://img.test/a.png" || r.ThumbnailURL != "https://ts.test/thumb.jpg" {
		t.Errorf("图片地址错误: %+v", r)
	}
	if r.Width != 800 || r.Height != 600 {
		t.Errorf("尺寸错误: %+v", r)
	}
}

func TestParseVideos(t *testing.T) {
	body := `<a class="mc_vtvc_link" href="https://video.test/watch?v=1" aria-label="Go talk · from youtube.com · Duration: 12 minutes · uploaded on 2 days ago · uploaded by Alice">
	<div class="mc_vtvc_title" title="Go Talk">Go Talk</div>
	<div class="mc_bc_rc items">12:00</div>
	<img data-src-hq="https://img.test/thumb.jpg"/></a>`
	page := parseVideos([]byte(body))
	if len(page.Results) != 1 {
		t.Fatalf("期望 1 个视频，实际 %d", len(page.Results))
	}
	r := page.Results[0]
	if r.Title != "Go Talk" || r.Source != "youtube.com" || r.Duration != "12:00" || r.Uploader != "Alice" {
		t.Errorf("视频字段错误: %+v", r)
	}
}

func TestBuildQuery(t *testing.T) {
	o := SearchOptions{
		Query: "http server", ExactMatch: true, Site: "go.dev", FileType: "pdf", Exclude: "npm yarn",
	}
	got := buildQuery(o)
	want := `"http server" site:go.dev filetype:pdf -npm -yarn`
	if got != want {
		t.Fatalf("buildQuery = %q, want %q", got, want)
	}
	// 已经带引号时不应重复添加
	o.Query = `"go"`
	if strings.Count(buildQuery(o), `"`) != 2 {
		t.Errorf("重复添加引号: %q", buildQuery(o))
	}
}

func TestBuildURLFeatures(t *testing.T) {
	u := buildURL("web", SearchOptions{
		Query: "golang", Count: 5, Offset: 10, Market: "en-US",
		SafeSearch: "strict", Freshness: "week", Extra: map[string]string{"form": "QBRE"},
	})
	for _, want := range []string{"first=11", "count=5", "mkt=en-US", "adlt=strict", `filters=ex1%3A%22ez2%22`, "form=QBRE"} {
		if !strings.Contains(u, want) {
			t.Errorf("URL 缺少 %q: %s", want, u)
		}
	}
}
