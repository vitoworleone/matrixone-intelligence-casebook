package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout      = 10 * time.Second
	defaultMaxDocumentChars = 4000
)

type OpenAIConfig struct {
	Endpoint         string
	APIKey           string
	Model            string
	ReturnDocuments  bool
	MaxDocumentChars int
	HTTPClient       *http.Client
}

type OpenAIReranker struct {
	endpoint         string
	apiKey           string
	model            string
	returnDocuments  bool
	maxDocumentChars int
	httpClient       *http.Client
}

func NewOpenAIReranker(cfg OpenAIConfig) (*OpenAIReranker, error) {
	endpoint := normalizeEndpoint(cfg.Endpoint)
	if endpoint == "" {
		return nil, errors.New("rerank endpoint is required")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("invalid rerank endpoint: %w", err)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("rerank model is required")
	}
	maxDocumentChars := cfg.MaxDocumentChars
	if maxDocumentChars <= 0 {
		maxDocumentChars = defaultMaxDocumentChars
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &OpenAIReranker{
		endpoint:         endpoint,
		apiKey:           strings.TrimSpace(cfg.APIKey),
		model:            model,
		returnDocuments:  cfg.ReturnDocuments,
		maxDocumentChars: maxDocumentChars,
		httpClient:       client,
	}, nil
}

func (r *OpenAIReranker) Rerank(ctx context.Context, req Request) (*Response, error) {
	if r == nil {
		return nil, errors.New("openai reranker is nil")
	}
	if len(req.Chunks) == 0 {
		return &Response{Strategy: r.strategy()}, nil
	}
	documents := make([]string, 0, len(req.Chunks))
	indexMap := make([]int, 0, len(req.Chunks))
	for i, chunk := range req.Chunks {
		content := truncateRunes(strings.TrimSpace(chunk.Content), r.maxDocumentChars)
		if content == "" {
			continue
		}
		documents = append(documents, content)
		indexMap = append(indexMap, i)
	}
	if len(documents) == 0 {
		return &Response{Strategy: r.strategy()}, nil
	}
	payload := openAIRequest{
		Model:           r.model,
		Query:           buildRerankQuery(req),
		Documents:       documents,
		TopN:            len(documents),
		ReturnDocuments: r.returnDocuments,
	}
	if strings.TrimSpace(payload.Query) == "" {
		return nil, errors.New("rerank query is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal rerank request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build rerank request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.apiKey)
	}
	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call rerank endpoint: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read rerank response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank endpoint status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var decoded openAIResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	scores := make([]Score, 0, len(decoded.Results))
	for _, result := range decoded.Results {
		if result.Index < 0 || result.Index >= len(indexMap) {
			continue
		}
		scores = append(scores, Score{
			Index: indexMap[result.Index],
			Score: result.RelevanceScore,
		})
	}
	return &Response{Strategy: r.strategy(), Scores: scores}, nil
}

func (r *OpenAIReranker) strategy() string {
	if r == nil || r.model == "" {
		return "openai"
	}
	return "openai:" + r.model
}

type openAIRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            int      `json:"top_n"`
	ReturnDocuments bool     `json:"return_documents"`
}

type openAIResponse struct {
	Results []openAIResult `json:"results"`
}

type openAIResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(endpoint, "/v1") {
		return endpoint + "/rerank"
	}
	if strings.HasSuffix(endpoint, "/rerank") {
		return endpoint
	}
	return endpoint
}

func buildRerankQuery(req Request) string {
	if query := strings.TrimSpace(req.Query); query != "" {
		return query
	}
	var parts []string
	for _, target := range req.Targets {
		parts = appendNonEmpty(parts, target.Description)
		for _, anchor := range target.Anchors {
			parts = appendNonEmpty(parts, anchor.Text)
		}
	}
	parts = append(parts, req.QueryHints...)
	parts = dedupe(parts)
	if len(parts) > 16 {
		parts = parts[:16]
	}
	return strings.Join(parts, " ")
}

func appendNonEmpty(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	return append(values, value)
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func truncateRunes(value string, max int) string {
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}
