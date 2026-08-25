package knowledge

import (
	"strings"
	"testing"
)

func TestStripInternalRAGChunkLocators(t *testing.T) {
	in := "基本信息：作者裴建锁 70b9db39-31e5-4d52-a58e-01159ddf7e8b:1:chunk:0:0。研究方法 [70b9db39-31e5-4d52-a58e-01159ddf7e8b:1:chunk:4:6]。"
	out := StripInternalRAGChunkLocators(in)
	if strings.Contains(out, "70b9db39") || strings.Contains(out, ":chunk:") {
		t.Fatalf("locator leaked: %q", out)
	}
	if !strings.Contains(out, "基本信息：作者裴建锁") || !strings.Contains(out, "研究方法") {
		t.Fatalf("prose lost: %q", out)
	}
}

func TestStripInternalRAGChunkLocatorsPreservesOtherWhitespace(t *testing.T) {
	in := "  before 70b9db39-31e5-4d52-a58e-01159ddf7e8b:1:chunk:0:0 after\n\n"
	if got, want := StripInternalRAGChunkLocators(in), "  before  after\n\n"; got != want {
		t.Fatalf("sanitized answer = %q, want %q", got, want)
	}
}
