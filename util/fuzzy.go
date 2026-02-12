package util

import (
	"strings"
)

type Match struct {
	Text  string
	Score int
	Index int
}

func FuzzyMatch(query string, candidates []string) []Match {
	if query == "" {
		return nil
	}

	if len(candidates) == 0 {
		return nil
	}

	query = strings.ToLower(query)
	matches := make([]Match, 0)

	for i, candidate := range candidates {
		if candidate == "" {
			continue
		}

		score := calculateScore(query, strings.ToLower(candidate))
		if score > 0 {
			matches = append(matches, Match{
				Text:  candidate,
				Score: score,
				Index: i,
			})
		}
	}

	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

func calculateScore(query string, text string) int {
	if query == "" {
		return 0
	}

	if text == "" {
		return 0
	}

	if strings.Contains(text, query) {
		return 100
	}

	score := 0
	queryIdx := 0
	consecutive := 0

	for i := 0; i < len(text); i++ {
		if queryIdx >= len(query) {
			break
		}

		if text[i] == query[queryIdx] {
			score += 10
			consecutive++
			if consecutive > 1 {
				score += consecutive * 2
			}
			queryIdx++
			continue
		}

		consecutive = 0
	}

	if queryIdx == len(query) {
		return score
	}

	return 0
}
