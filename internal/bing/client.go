package bing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const maxBodyBytes = 8 << 20 // 8 MB，防止异常响应耗尽内存

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// Client 负责请求 Bing 并处理重试、超时与会话 Cookie。
type Client struct {
	http    *http.Client
	retries int
}

// NewClient 创建一个 Bing 客户端。timeout<=0 时使用 15 秒。
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{
			Timeout: timeout,
			Jar:     jar,
			Transport: &http.Transport{
				MaxIdleConns:        64,
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		retries: 2,
	}
}

// get 发起 GET 请求，对 429/5xx 和临时网络错误做指数退避重试。
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build request: %w", err)
		}
		req.Header.Set("User-Agent", userAgents[attempt%len(userAgents)])
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8")
		req.Header.Set("Referer", baseURL+"/")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if !c.sleepBackoff(ctx, attempt) {
				return nil, ctx.Err()
			}
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if !c.sleepBackoff(ctx, attempt) {
				return nil, ctx.Err()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		// 204 No Content：联想词接口在无结果时会返回，视为空响应而非错误。
		if resp.StatusCode == http.StatusNoContent {
			return nil, nil
		}
		lastErr = fmt.Errorf("Bing returned HTTP %d (%s)", resp.StatusCode, rawURL)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if !c.sleepBackoff(ctx, attempt) {
				return nil, ctx.Err()
			}
			continue
		}
		return nil, lastErr
	}
	return nil, fmt.Errorf("Bing request failed after %d retries: %w", c.retries, lastErr)
}

func (c *Client) sleepBackoff(ctx context.Context, attempt int) bool {
	delay := time.Duration(attempt+1) * 400 * time.Millisecond
	if attempt == c.retries {
		return false
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
