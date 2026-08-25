package rerank

type Request struct {
	Query      string
	Targets    []Target
	QueryHints []string
	Chunks     []CandidateChunk
}

type Response struct {
	Strategy string
	Scores   []Score
}

type Score struct {
	Index int
	Score float64
}

type Target struct {
	Description string
	Anchors     []Anchor
}

type Anchor struct {
	Text string
}

type CandidateChunk struct {
	Content string
}
