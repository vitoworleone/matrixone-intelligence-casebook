package knowledge

import (
	"regexp"
)

// rawRAGChunkLocatorRe matches internal evidence locators that must not appear
// in user-facing answers, e.g. 70b9db39-...:1:chunk:0:0 (#12968).
var rawRAGChunkLocatorRe = regexp.MustCompile(`(?i)(?:\[|\()?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}:\d+:chunk:\d+:\d+(?:\]|\))?`)

var rawRAGChunkShortRe = regexp.MustCompile(`(?i)(?:\[|\()?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}:chunk:\d+(?::\d+)?(?:\]|\))?`)

// StripInternalRAGChunkLocators removes internal RAG chunk locator tokens from
// user-facing answer text while keeping natural language content.
func StripInternalRAGChunkLocators(answer string) string {
	next := rawRAGChunkLocatorRe.ReplaceAllString(answer, "")
	return rawRAGChunkShortRe.ReplaceAllString(next, "")
}
