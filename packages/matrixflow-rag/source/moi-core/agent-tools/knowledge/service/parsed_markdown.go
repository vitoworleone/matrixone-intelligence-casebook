package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

const (
	defaultParsedMarkdownLimitChars   = 8000
	maxParsedMarkdownLimitChars       = 20000
	defaultParsedMarkdownMaxMatches   = 5
	maxParsedMarkdownMaxMatches       = 20
	defaultParsedMarkdownContextChars = 600
	maxParsedMarkdownContextChars     = 2000
)

type ParsedMarkdownBackend interface {
	LoadParsedMarkdown(ctx context.Context, scope knowledge.WorkspaceScope, markdownFileID string) (*ParsedMarkdownDocument, error)
}

type ParsedMarkdownDocument struct {
	Content  string
	FileName string
	Metadata map[string]any
}

type readParsedMarkdownService struct {
	backend ParsedMarkdownBackend
}

type searchParsedMarkdownService struct {
	backend ParsedMarkdownBackend
}

func NewReadParsedMarkdown(deps Deps) knowledge.ReadParsedMarkdown {
	return &readParsedMarkdownService{backend: deps.ParsedMarkdownBackend}
}

func NewSearchParsedMarkdown(deps Deps) knowledge.SearchParsedMarkdown {
	return &searchParsedMarkdownService{backend: deps.ParsedMarkdownBackend}
}

func (s *readParsedMarkdownService) Execute(ctx context.Context, req knowledge.ReadParsedMarkdownRequest) (*knowledge.ReadParsedMarkdownResponse, error) {
	if s == nil || s.backend == nil {
		return nil, fmt.Errorf("read parsed markdown: backend is not configured")
	}
	markdownFileID := strings.TrimSpace(req.MarkdownFileID)
	if markdownFileID == "" {
		return nil, fmt.Errorf("read parsed markdown: markdown_file_id is required")
	}
	if req.Cursor < 0 {
		return nil, fmt.Errorf("read parsed markdown: cursor must be >= 0")
	}
	limitChars := normalizeParsedMarkdownLimitChars(req.LimitChars)
	if limitChars > maxParsedMarkdownLimitChars {
		return nil, fmt.Errorf("read parsed markdown: limit_chars must be <= %d", maxParsedMarkdownLimitChars)
	}
	doc, err := s.backend.LoadParsedMarkdown(ctx, req.Scope, markdownFileID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("read parsed markdown: backend returned nil document")
	}
	runes := []rune(doc.Content)
	if req.Cursor > len(runes) {
		return nil, fmt.Errorf("read parsed markdown: cursor %d exceeds total chars %d", req.Cursor, len(runes))
	}
	end := req.Cursor + limitChars
	if end > len(runes) {
		end = len(runes)
	}
	nextCursor := 0
	if end < len(runes) {
		nextCursor = end
	}
	return &knowledge.ReadParsedMarkdownResponse{
		MarkdownFileID: markdownFileID,
		FileName:       doc.FileName,
		TotalChars:     len(runes),
		Cursor:         req.Cursor,
		ReturnedChars:  end - req.Cursor,
		NextCursor:     nextCursor,
		Content:        string(runes[req.Cursor:end]),
		Metadata:       doc.Metadata,
	}, nil
}

func (s *searchParsedMarkdownService) Execute(ctx context.Context, req knowledge.SearchParsedMarkdownRequest) (*knowledge.SearchParsedMarkdownResponse, error) {
	if s == nil || s.backend == nil {
		return nil, fmt.Errorf("search parsed markdown: backend is not configured")
	}
	markdownFileID := strings.TrimSpace(req.MarkdownFileID)
	if markdownFileID == "" {
		return nil, fmt.Errorf("search parsed markdown: markdown_file_id is required")
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("search parsed markdown: query is required")
	}
	maxMatches := normalizeParsedMarkdownMaxMatches(req.MaxMatches)
	if maxMatches > maxParsedMarkdownMaxMatches {
		return nil, fmt.Errorf("search parsed markdown: max_matches must be <= %d", maxParsedMarkdownMaxMatches)
	}
	contextChars := normalizeParsedMarkdownContextChars(req.ContextChars)
	if contextChars > maxParsedMarkdownContextChars {
		return nil, fmt.Errorf("search parsed markdown: context_chars must be <= %d", maxParsedMarkdownContextChars)
	}
	doc, err := s.backend.LoadParsedMarkdown(ctx, req.Scope, markdownFileID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("search parsed markdown: backend returned nil document")
	}
	runes := []rune(doc.Content)
	queryRunes := []rune(query)
	matches := make([]knowledge.MarkdownMatch, 0, maxMatches)
	offset := 0
	totalMatches := 0
	for offset <= len(runes) {
		idx := runeSliceIndex(runes[offset:], queryRunes)
		if idx < 0 {
			break
		}
		start := offset + idx
		end := start + len(queryRunes)
		totalMatches++
		if len(matches) < maxMatches {
			snippetStart := start - contextChars
			if snippetStart < 0 {
				snippetStart = 0
			}
			snippetEnd := end + contextChars
			if snippetEnd > len(runes) {
				snippetEnd = len(runes)
			}
			matches = append(matches, knowledge.MarkdownMatch{
				Start:   start,
				End:     end,
				Snippet: string(runes[snippetStart:snippetEnd]),
			})
		}
		offset = end
	}
	return &knowledge.SearchParsedMarkdownResponse{
		MarkdownFileID: markdownFileID,
		FileName:       doc.FileName,
		Query:          query,
		TotalChars:     len(runes),
		Matches:        matches,
		TotalMatches:   totalMatches,
		Truncated:      totalMatches > len(matches),
		Metadata:       doc.Metadata,
	}, nil
}

func normalizeParsedMarkdownLimitChars(limit int) int {
	if limit <= 0 {
		return defaultParsedMarkdownLimitChars
	}
	return limit
}

func normalizeParsedMarkdownMaxMatches(maxMatches int) int {
	if maxMatches <= 0 {
		return defaultParsedMarkdownMaxMatches
	}
	return maxMatches
}

func normalizeParsedMarkdownContextChars(contextChars int) int {
	if contextChars <= 0 {
		return defaultParsedMarkdownContextChars
	}
	return contextChars
}

func runeSliceIndex(haystack []rune, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	if len(needle) > len(haystack) {
		return -1
	}
	for idx := 0; idx <= len(haystack)-len(needle); idx++ {
		matched := true
		for needleIdx := range needle {
			if haystack[idx+needleIdx] != needle[needleIdx] {
				matched = false
				break
			}
		}
		if matched {
			return idx
		}
	}
	return -1
}
