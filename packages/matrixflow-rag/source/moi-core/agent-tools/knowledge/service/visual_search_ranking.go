package service

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

const (
	visualRankingProfileDocument             = "document"
	visualTextRegionPreScoreWeight           = 0.1
	visualTextRegionSameObjectBonusThreshold = 2
	visualTextRegionSameObjectBonus          = 0.45
	visualTextRegionNearTieGap               = 1
)

func visualRankingProfile(profile string) string {
	switch strings.TrimSpace(profile) {
	case "":
		return visualRankingProfileDocument
	case knowledge.VisualSearchRankingProfileVisualObjectFirst:
		return knowledge.VisualSearchRankingProfileVisualObjectFirst
	case knowledge.VisualSearchRankingProfileTextRegionFirst:
		return knowledge.VisualSearchRankingProfileTextRegionFirst
	default:
		return strings.TrimSpace(profile)
	}
}

func visualRankingProfileValid(profile string) bool {
	return profile == visualRankingProfileDocument ||
		profile == knowledge.VisualSearchRankingProfileVisualObjectFirst ||
		profile == knowledge.VisualSearchRankingProfileTextRegionFirst
}

func fuseVisualResults(textHits, imageHits []knowledge.VisualSearchHit, topK int, rankingProfile string) []knowledge.VisualSearchHit {
	rankingProfile = visualRankingProfile(rankingProfile)
	type item struct {
		hit        knowledge.VisualSearchHit
		textScore  float64
		imageScore float64
		fusedScore float64
		textRank   int
		imageRank  int
		textHits   int
		imageHits  int
	}
	byKey := map[string]*item{}
	add := func(hits []knowledge.VisualSearchHit, image bool) {
		for idx, hit := range hits {
			key := visualFusionKey(hit, rankingProfile)
			if key == "" {
				continue
			}
			current := byKey[key]
			if current == nil {
				copyHit := hit
				current = &item{hit: copyHit}
				byKey[key] = current
			}
			score := 1.0 / float64(60+idx+1)
			current.fusedScore += score
			if image {
				current.imageHits++
				if current.imageRank == 0 || idx+1 < current.imageRank {
					current.imageRank = idx + 1
				}
				if hit.Score > current.imageScore {
					current.imageScore = hit.Score
				}
				if hit.Score > current.hit.Score {
					current.hit = hit
				}
			} else {
				current.textHits++
				if current.textRank == 0 || idx+1 < current.textRank {
					current.textRank = idx + 1
				}
				if hit.Score > current.textScore {
					current.textScore = hit.Score
				}
				if current.imageRank == 0 && hit.Score >= current.hit.Score {
					current.hit = hit
				}
			}
		}
	}
	add(textHits, false)
	add(imageHits, true)
	items := make([]*item, 0, len(byKey))
	for _, current := range byKey {
		items = append(items, current)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].fusedScore == items[j].fusedScore {
			return items[i].hit.ObjectID < items[j].hit.ObjectID
		}
		return items[i].fusedScore > items[j].fusedScore
	})
	if topK <= 0 {
		topK = defaultVisualTopK
	}
	out := make([]knowledge.VisualSearchHit, 0, minInt(topK, len(items)))
	for _, current := range items {
		hit := current.hit
		hit.Score = current.fusedScore
		fusionKeyType := "document"
		reason := "document-level hybrid RRF match across text and image visual indexes"
		if visualRankingProfileUsesObjectScope(rankingProfile) {
			fusionKeyType = "visual_object"
			reason = "object-level hybrid RRF match across text and image visual object indexes"
		}
		hit.ScoreParts = map[string]any{
			"text":            current.textScore,
			"image":           current.imageScore,
			"fused":           current.fusedScore,
			"text_rank":       current.textRank,
			"image_rank":      current.imageRank,
			"text_hit_count":  current.textHits,
			"image_hit_count": current.imageHits,
			"fusion_key_type": fusionKeyType,
			"fusion_key":      visualFusionKey(hit, rankingProfile),
		}
		if visualRankingProfileUsesObjectScope(rankingProfile) {
			hit.ScoreParts["ranking_profile"] = rankingProfile
			hit.ScoreParts["scope"] = hit.Scope
		}
		hit.Reason = reason
		out = append(out, hit)
		if len(out) >= topK {
			break
		}
	}
	return out
}

func visualFusionKey(hit knowledge.VisualSearchHit, rankingProfile string) string {
	if visualRankingProfileUsesObjectScope(rankingProfile) {
		return visualObjectFusionKey(hit)
	}
	base := firstNonEmpty(hit.SourceFileID, fmt.Sprintf("%s:%d:%v", hit.SourceFileID, hit.PageNumber, hit.BBox), hit.ObjectID, hit.ImageFileID, hit.PageImageFileID)
	return visualSemanticModelEvidenceKey(hit.SemanticModelID, base)
}

func visualRankingProfileUsesObjectScope(rankingProfile string) bool {
	switch visualRankingProfile(rankingProfile) {
	case knowledge.VisualSearchRankingProfileVisualObjectFirst, knowledge.VisualSearchRankingProfileTextRegionFirst:
		return true
	default:
		return false
	}
}

func visualObjectFusionKey(hit knowledge.VisualSearchHit) string {
	if strings.TrimSpace(hit.SourceFileID) != "" && strings.TrimSpace(hit.ObjectID) != "" {
		return visualSemanticModelEvidenceKey(hit.SemanticModelID, strings.TrimSpace(hit.SourceFileID)+":"+strings.TrimSpace(hit.ObjectID))
	}
	base := firstNonEmpty(hit.ImageFileID, hit.PageImageFileID, fmt.Sprintf("%s:%d:%v", hit.SourceFileID, hit.PageNumber, hit.BBox))
	return visualSemanticModelEvidenceKey(hit.SemanticModelID, base)
}

func rerankVisualObjectFirstResults(hits []knowledge.VisualSearchHit, constraints []string, topK int) []knowledge.VisualSearchHit {
	constraints = visualEngineeringConstraints(constraints)
	if len(hits) == 0 || len(constraints) == 0 {
		return hits
	}
	type rankedHit struct {
		hit   knowledge.VisualSearchHit
		score float64
		index int
	}
	ranked := make([]rankedHit, 0, len(hits))
	for idx, hit := range hits {
		matches := visualConstraintMatches(hit.Content, constraints)
		matchRatio := float64(len(matches)) / float64(len(constraints))
		sameObjectBonus := 0.0
		if len(matches) >= 2 {
			sameObjectBonus = 0.35
		}
		objectKindPenalty := visualObjectKindPenalty(hit)
		finalScore := hit.Score + 1.0*matchRatio + sameObjectBonus - objectKindPenalty
		scoreParts := cloneMap(hit.ScoreParts)
		scoreParts["constraint_match_count"] = len(matches)
		scoreParts["constraint_total_count"] = len(constraints)
		scoreParts["constraint_match_ratio"] = matchRatio
		scoreParts["constraint_matches"] = append([]string(nil), matches...)
		scoreParts["same_object_constraint_bonus"] = sameObjectBonus
		scoreParts["object_kind_penalty"] = objectKindPenalty
		scoreParts["pre_constraint_score"] = hit.Score
		scoreParts["final_score"] = finalScore
		hit.ScoreParts = scoreParts
		hit.Score = finalScore
		if len(matches) > 0 {
			hit.Reason = hit.Reason + "; reranked by visual object text constraints"
		}
		ranked = append(ranked, rankedHit{hit: hit, score: finalScore, index: idx})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].index < ranked[j].index
		}
		return ranked[i].score > ranked[j].score
	})
	if topK <= 0 {
		topK = defaultVisualTopK
	}
	out := make([]knowledge.VisualSearchHit, 0, minInt(topK, len(ranked)))
	for _, item := range ranked {
		out = append(out, item.hit)
		if len(out) >= topK {
			break
		}
	}
	return out
}

func rerankVisualTextRegionFirstResults(hits []knowledge.VisualSearchHit, fragments []string, topK int) []knowledge.VisualSearchHit {
	fragments = visualTextRegionFragments(fragments)
	if len(hits) == 0 || len(fragments) == 0 {
		return hits
	}
	ranked := make([]rankedVisualTextRegionHit, 0, len(hits))
	groupHitCounts := map[string]int{}
	for _, hit := range hits {
		if groupKey := visualTextRegionGroupKey(hit); groupKey != "" {
			groupHitCounts[groupKey]++
		}
	}
	maxMatchCount := 0
	for idx, hit := range hits {
		matches := visualConstraintMatches(hit.Content, fragments)
		if len(matches) > maxMatchCount {
			maxMatchCount = len(matches)
		}
		matchRatio := float64(len(matches)) / float64(len(fragments))
		sameObjectBonus := 0.0
		if len(matches) >= visualTextRegionSameObjectBonusThreshold {
			sameObjectBonus = visualTextRegionSameObjectBonus
		}
		textSignalScore := float64(len(matches)) + matchRatio + sameObjectBonus
		finalScore := hit.Score*visualTextRegionPreScoreWeight + textSignalScore
		scoreParts := cloneMap(hit.ScoreParts)
		imageRank := visualTextRegionImageRank(scoreParts)
		groupKey := visualTextRegionGroupKey(hit)
		scoreParts["text_region_match_count"] = len(matches)
		scoreParts["text_region_total_count"] = len(fragments)
		scoreParts["text_region_match_ratio"] = matchRatio
		scoreParts["text_region_matches"] = append([]string(nil), matches...)
		scoreParts["same_object_text_region_bonus"] = sameObjectBonus
		scoreParts["pre_text_region_score"] = hit.Score
		scoreParts["final_score"] = finalScore
		scoreParts["text_region_image_rank"] = imageRank
		scoreParts["text_region_group_key"] = groupKey
		scoreParts["text_region_group_hit_count"] = groupHitCounts[groupKey]
		scoreParts["text_region_visual_tiebreak_applied"] = false
		hit.ScoreParts = scoreParts
		hit.Score = finalScore
		if len(matches) > 0 {
			hit.Reason = hit.Reason + "; reranked by visual text/table region constraints"
		}
		ranked = append(ranked, rankedVisualTextRegionHit{hit: hit, score: finalScore, matchCount: len(matches), imageRank: imageRank, index: idx})
	}
	for idx := range ranked {
		ranked[idx].nearTie = maxMatchCount-ranked[idx].matchCount <= visualTextRegionNearTieGap
		ranked[idx].key = textRegionRankKey{
			matchCount: ranked[idx].matchCount,
			nearTie:    ranked[idx].nearTie,
			imageRank:  ranked[idx].imageRank,
			score:      ranked[idx].score,
			index:      ranked[idx].index,
		}
		ranked[idx].hit.ScoreParts["text_region_near_tie"] = ranked[idx].nearTie
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return lessTextRegionRankKey(ranked[i].key, ranked[j].key)
	})
	if topK <= 0 {
		topK = defaultVisualTopK
	}
	out := make([]knowledge.VisualSearchHit, 0, minInt(topK, len(ranked)))
	if len(ranked) > 0 {
		ranked[0].hit.ScoreParts["text_region_visual_tiebreak_applied"] = textRegionVisualTiebreakApplied(ranked[0].key, maxMatchCount)
	}
	for _, item := range ranked {
		out = append(out, item.hit)
		if len(out) >= topK {
			break
		}
	}
	return out
}

type rankedVisualTextRegionHit struct {
	hit        knowledge.VisualSearchHit
	score      float64
	matchCount int
	imageRank  int
	nearTie    bool
	index      int
	key        textRegionRankKey
}

type textRegionRankKey struct {
	matchCount int
	nearTie    bool
	imageRank  int
	score      float64
	index      int
}

func lessTextRegionRankKey(left, right textRegionRankKey) bool {
	if left.nearTie && right.nearTie && left.matchCount > 0 && right.matchCount > 0 {
		leftHasImage := left.imageRank > 0
		rightHasImage := right.imageRank > 0
		if leftHasImage != rightHasImage {
			return leftHasImage
		}
		if leftHasImage && left.imageRank != right.imageRank {
			return left.imageRank < right.imageRank
		}
	}
	if left.score != right.score {
		return left.score > right.score
	}
	return left.index < right.index
}

func textRegionVisualTiebreakApplied(key textRegionRankKey, maxMatchCount int) bool {
	return key.nearTie && key.imageRank > 0 && key.matchCount > 0 && key.matchCount < maxMatchCount
}

func visualTextRegionImageRank(scoreParts map[string]any) int {
	if scoreParts == nil {
		return 0
	}
	if value, ok := visualTextRegionIntFromAny(scoreParts["image_rank"]); ok {
		return value
	}
	return 0
}

func visualTextRegionGroupKey(hit knowledge.VisualSearchHit) string {
	if strings.TrimSpace(hit.SourceFileID) != "" {
		return visualSemanticModelEvidenceKey(hit.SemanticModelID, strings.TrimSpace(hit.SourceFileID))
	}
	return visualSemanticModelEvidenceKey(hit.SemanticModelID, strings.TrimSpace(hit.SourceFileName))
}

func visualSemanticModelEvidenceKey(semanticModelID int64, evidenceKey string) string {
	evidenceKey = strings.TrimSpace(evidenceKey)
	if evidenceKey == "" || semanticModelID <= 0 {
		return evidenceKey
	}
	return fmt.Sprintf("semantic_model:%d:%s", semanticModelID, evidenceKey)
}

func visualTextRegionIntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(math.Round(float64(typed))), true
	case float64:
		return int(math.Round(typed)), true
	default:
		return 0, false
	}
}

func visualTextRegionFragments(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func visualEngineeringConstraints(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range compactStrings(values) {
		if visualConstraintIsDiscriminative(value) {
			out = append(out, value)
		}
	}
	if len(out) > 0 {
		return out
	}
	return compactStrings(values)
}

func visualConstraintIsDiscriminative(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "0123456789") {
		return true
	}
	for _, marker := range []string{"Ø", "Φ", "±", "+", "-", "−", "°"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func visualConstraintMatches(content string, constraints []string) []string {
	if strings.TrimSpace(content) == "" || len(constraints) == 0 {
		return nil
	}
	matches := make([]string, 0, len(constraints))
	for _, constraint := range constraints {
		constraint = strings.TrimSpace(constraint)
		if constraint == "" {
			continue
		}
		if strings.Contains(content, constraint) {
			matches = append(matches, constraint)
		}
	}
	return matches
}

func visualObjectKindPenalty(hit knowledge.VisualSearchHit) float64 {
	switch strings.TrimSpace(hit.ObjectKind) {
	case "title_block", "notes":
		return 0.7
	default:
		return 0
	}
}

func cloneMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}
