package rerank

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIRerankerCallsEndpointAndMapsScores(t *testing.T) {
	var captured openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"index": 1, "relevance_score": 0.91},
				{"index": 0, "relevance_score": 0.12},
			},
		})
	}))
	defer server.Close()

	client, err := NewOpenAIReranker(OpenAIConfig{
		Endpoint: server.URL + "/v1/rerank",
		APIKey:   "token-1",
		Model:    "bge-reranker-v2-m3",
	})
	if err != nil {
		t.Fatalf("new reranker: %v", err)
	}
	resp, err := client.Rerank(context.Background(), Request{
		Query: "2024 营业收入 同比增长",
		Chunks: []CandidateChunk{
			{Content: "标题"},
			{Content: "2024年度，公司实现营业收入69.01亿元，较上年同期增长3.50%"},
		},
	})
	if err != nil {
		t.Fatalf("rerank: %v", err)
	}
	if captured.Model != "bge-reranker-v2-m3" || captured.Query != "2024 营业收入 同比增长" {
		t.Fatalf("captured request = %+v", captured)
	}
	if captured.TopN != 2 || len(captured.Documents) != 2 {
		t.Fatalf("captured request = %+v", captured)
	}
	if resp.Strategy != "openai:bge-reranker-v2-m3" {
		t.Fatalf("strategy = %q", resp.Strategy)
	}
	if len(resp.Scores) != 2 || resp.Scores[0].Index != 1 || resp.Scores[0].Score != 0.91 {
		t.Fatalf("scores = %+v", resp.Scores)
	}
}

func TestOpenAIRerankerBuildsQueryFromTargets(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		query = req.Query
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"index": 0, "relevance_score": 0.5}},
		})
	}))
	defer server.Close()
	client, err := NewOpenAIReranker(OpenAIConfig{Endpoint: server.URL + "/v1", Model: "ranker"})
	if err != nil {
		t.Fatalf("new reranker: %v", err)
	}
	_, err = client.Rerank(context.Background(), Request{
		Targets: []Target{{
			Description: "2022 营业收入同比增长",
			Anchors: []Anchor{
				{Text: "2022"},
				{Text: "营业收入"},
			},
		}},
		QueryHints: []string{"较上年同期增长"},
		Chunks:     []CandidateChunk{{Content: "2022年营业收入增长"}},
	})
	if err != nil {
		t.Fatalf("rerank: %v", err)
	}
	for _, want := range []string{"2022", "营业收入", "较上年同期增长"} {
		if !strings.Contains(query, want) {
			t.Fatalf("query %q missing %q", query, want)
		}
	}
}

func TestOpenAIRerankerReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()
	client, err := NewOpenAIReranker(OpenAIConfig{Endpoint: server.URL + "/v1/rerank", Model: "ranker"})
	if err != nil {
		t.Fatalf("new reranker: %v", err)
	}
	_, err = client.Rerank(context.Background(), Request{
		Query:  "query",
		Chunks: []CandidateChunk{{Content: "doc"}},
	})
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected provider status error, got %v", err)
	}
}
