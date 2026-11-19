package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"bing-search-api/internal/bing"
)

type searchRequest struct {
	Q          string            `json:"q"`
	Query      string            `json:"query"`
	Count      int               `json:"count"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	Page       int               `json:"page"`
	Language   string            `json:"language"`
	Lang       string            `json:"lang"`
	Region     string            `json:"region"`
	Market     string            `json:"market"`
	SafeSearch string            `json:"safe_search"`
	Safe       string            `json:"safe"`
	Freshness  string            `json:"freshness"`
	Site       string            `json:"site"`
	FileType   string            `json:"file_type"`
	Exclude    string            `json:"exclude"`
	ExactMatch bool              `json:"exact_match"`
	Filters    string            `json:"filters"`
	QFT        string            `json:"qft"`
	Extra      map[string]string `json:"extra"`
}

// response 是所有接口统一返回的 JSON 结构。
type response struct {
	Success        bool        `json:"success"`
	Type           string      `json:"type,omitempty"`
	Query          string      `json:"query,omitempty"`
	Count          int         `json:"count"`
	Offset         int         `json:"offset"`
	TotalEstimated string      `json:"total_estimated,omitempty"`
	Results        interface{} `json:"results,omitempty"`
	Data           interface{} `json:"data,omitempty"`
	RequestURL     string      `json:"request_url,omitempty"`
	Fallback       bool        `json:"fallback,omitempty"`
	Note           string      `json:"note,omitempty"`
	Error          string      `json:"error,omitempty"`
	RequestID      string      `json:"request_id"`
}

func (s *Server) handleVertical(vertical string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts, err := parseOptions(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		if err := opts.Validate(); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		ctx, cancel := withTimeout(r)
		defer cancel()

		var resp response
		switch vertical {
		case "web":
			page, err := s.client.Web(ctx, *opts)
			if err != nil {
				writeUpstreamError(w, r, err)
				return
			}
			resp = response{Success: true, Type: "web", Query: page.Query, Count: len(page.Results), Offset: opts.Offset,
				TotalEstimated: page.TotalEstimated, Results: page.Results, RequestURL: page.RequestURL}
		case "images":
			page, err := s.client.Images(ctx, *opts)
			if err != nil {
				writeUpstreamError(w, r, err)
				return
			}
			resp = response{Success: true, Type: "images", Query: page.Query, Count: len(page.Results), Offset: opts.Offset,
				TotalEstimated: page.TotalEstimated, Results: page.Results, RequestURL: page.RequestURL}
		case "videos":
			page, err := s.client.Videos(ctx, *opts)
			if err != nil {
				writeUpstreamError(w, r, err)
				return
			}
			resp = response{Success: true, Type: "videos", Query: page.Query, Count: len(page.Results), Offset: opts.Offset,
				TotalEstimated: page.TotalEstimated, Results: page.Results, RequestURL: page.RequestURL}
		case "news":
			page, err := s.client.News(ctx, *opts)
			if err != nil {
				writeUpstreamError(w, r, err)
				return
			}
			resp = response{Success: true, Type: "news", Query: page.Query, Count: len(page.Results), Offset: opts.Offset,
				TotalEstimated: page.TotalEstimated, Results: page.Results, RequestURL: page.RequestURL,
				Fallback: page.Fallback, Note: page.Note}
		}
		writeJSON(w, r, http.StatusOK, resp)
	}
}

func (s *Server) handleAll(w http.ResponseWriter, r *http.Request) {
	opts, err := parseOptions(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := opts.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	verticals := []string{"web", "images", "videos", "news"}
	if v := strings.TrimSpace(r.URL.Query().Get("types")); v != "" {
		verticals = strings.FieldsFunc(v, func(c rune) bool { return c == ',' || c == ' ' })
	}
	for _, v := range verticals {
		switch v {
		case "web", "images", "videos", "news", "suggest":
		default:
			writeError(w, r, http.StatusBadRequest, fmt.Sprintf("unsupported search type: %s", v))
			return
		}
	}

	ctx, cancel := withTimeout(r)
	defer cancel()
	data, err := s.client.All(ctx, *opts, verticals)
	if err != nil {
		writeUpstreamError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, response{
		Success: true, Type: "all", Query: opts.Query, Count: len(data), Data: data,
	})
}

func (s *Server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	q := first(r.URL.Query().Get("q"), r.URL.Query().Get("query"))
	market := first(r.URL.Query().Get("market"), r.URL.Query().Get("mkt"))
	if q == "" && r.Method == http.MethodPost {
		var body struct {
			Q      string `json:"q"`
			Market string `json:"market"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		q, market = body.Q, first(body.Market, market)
	}
	if strings.TrimSpace(q) == "" {
		writeError(w, r, http.StatusBadRequest, "missing required parameter: q")
		return
	}
	ctx, cancel := withTimeout(r)
	defer cancel()
	items, err := s.client.Suggest(ctx, q, market)
	if err != nil {
		writeUpstreamError(w, r, err)
		return
	}
	resp := response{Success: true, Type: "suggest", Query: q, Count: len(items), Results: items}
	if len(items) == 0 {
		resp.Note = "Bing suggestions returned no results from the current network (common for datacenter IPs). The endpoint itself works; retry from a different network."
	}
	writeJSON(w, r, http.StatusOK, resp)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"status": "ok", "service": "bing-search-api",
	})
}

// parseOptions 同时兼容 URL 查询参数与 JSON 请求体。
func parseOptions(r *http.Request) (*bing.SearchOptions, error) {
	var body searchRequest
	if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&body); err != nil && err.Error() != "EOF" {
			return nil, fmt.Errorf("failed to parse JSON request body: %w", err)
		}
	}
	q := r.URL.Query()
	get := func(names ...string) string {
		for _, n := range names {
			if v := q.Get(n); v != "" {
				return v
			}
		}
		return ""
	}

	count, _ := strconv.Atoi(first(get("count", "limit"), strconv.Itoa(body.Count), strconv.Itoa(body.Limit)))
	offset, _ := strconv.Atoi(first(get("offset"), strconv.Itoa(body.Offset)))
	pageNo, _ := strconv.Atoi(first(get("page"), strconv.Itoa(body.Page)))
	if pageNo > 0 {
		if count <= 0 {
			count = 10
		}
		offset = (pageNo - 1) * count
	}
	exact := parseBool(first(get("exact_match", "exact"), strconv.FormatBool(body.ExactMatch)))

	opts := &bing.SearchOptions{
		Query:      first(get("q", "query"), body.Q, body.Query),
		Count:      count,
		Offset:     offset,
		Language:   first(get("language", "lang", "setlang"), body.Language, body.Lang),
		Region:     first(get("region", "cc", "country"), body.Region),
		Market:     first(get("market", "mkt"), body.Market),
		SafeSearch: first(get("safe_search", "safe", "adlt"), body.SafeSearch, body.Safe),
		Freshness:  first(get("freshness", "time"), body.Freshness),
		Site:       first(get("site"), body.Site),
		FileType:   first(get("file_type", "filetype"), body.FileType),
		Exclude:    first(get("exclude", "exclude_words"), body.Exclude),
		ExactMatch: exact,
		Filters:    first(get("filters"), body.Filters),
		QFT:        first(get("qft"), body.QFT),
		Extra:      body.Extra,
	}
	if opts.Extra == nil {
		opts.Extra = map[string]string{}
	}
	// 允许通过 b_ 前缀的查询参数透传任意 Bing 原生参数，例如 b_form=QBRE。
	for key, values := range q {
		if strings.HasPrefix(key, "b_") && len(values) > 0 {
			opts.Extra[strings.TrimPrefix(key, "b_")] = values[0]
		}
	}
	// extra JSON 参数（对象形式）
	if raw := get("extra"); raw != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			for k, v := range m {
				opts.Extra[k] = v
			}
		}
	}
	return opts, nil
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	switch x := v.(type) {
	case response:
		if x.RequestID == "" {
			x.RequestID = requestIDFrom(r.Context())
			v = x
		}
	case *response:
		if x.RequestID == "" {
			x.RequestID = requestIDFrom(r.Context())
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	writeJSON(w, r, status, response{Success: false, Error: msg, RequestID: requestIDFrom(r.Context())})
}

func writeUpstreamError(w http.ResponseWriter, r *http.Request, err error) {
	writeJSON(w, r, http.StatusBadGateway, response{
		Success: false, Error: "upstream Bing request failed: " + err.Error(), RequestID: requestIDFrom(r.Context()),
	})
}
