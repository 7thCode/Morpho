package viterbi

import (
	"github.com/7thCode/morpho/internal/hmm"
	"github.com/7thCode/morpho/internal/tokenizer"
)

// Result holds the decoded surface form, POS tag, and Viterbi score for a token.
type Result struct {
	Surface string
	POS     string
	Score   float64
}

// Decode runs the Viterbi algorithm over the given tokens using the HMM model.
// It returns a slice of Results with the most likely POS tag for each token.
//
// A token with a valid Tag (e.g. from a user-registered word) is pinned to
// that POS: the path is forced through it when the model has that state, and
// the reported POS is the tag either way. Decode treats tokens as a single
// sequence; split text into sentences first (hmm.SplitSentences).
func Decode(tokens []tokenizer.Token, model *hmm.Model) []Result {
	if len(tokens) == 0 || model == nil || len(model.POSTags) == 0 {
		return nil
	}

	T := len(tokens)
	S := len(model.POSTags)

	// dp[t][s] = max log-prob of best path to token t in state s
	dp := make([][]float64, T)
	// bp[t][s] = best previous state index at time t
	bp := make([][]int, T)

	for t := 0; t < T; t++ {
		dp[t] = make([]float64, S)
		bp[t] = make([]int, S)
		for s := 0; s < S; s++ {
			dp[t][s] = hmm.LogZero
			bp[t][s] = 0
		}
	}

	// forced[t] is the state token t is pinned to. Only a tag
	// naming a state of this model pins the token; other tags are applied
	// when results are built.
	forced := make([]string, T)
	for t, tok := range tokens {
		if hmm.IsValidPOS(tok.Tag) && model.HasPOS(tok.Tag) {
			forced[t] = tok.Tag
		}
	}
	allowed := func(t int, pos string) bool { return forced[t] == "" || forced[t] == pos }

	// Initialize: t=0. Models without a smoothed initial distribution may
	// lack an entry for a state; score it by emission alone rather than
	// pruning it, as the recursion does for missing transitions.
	for s, pos := range model.POSTags {
		if !allowed(0, pos) {
			continue
		}
		logInit := model.LogInitial(pos)
		if logInit <= hmm.LogZero {
			logInit = 0
		}
		dp[0][s] = logInit + model.SmoothEmission(pos, tokens[0].Surface)
	}

	// Recursion
	for t := 1; t < T; t++ {
		for s, pos := range model.POSTags {
			if !allowed(t, pos) {
				continue
			}
			logEmit := model.SmoothEmission(pos, tokens[t].Surface)
			bestScore := hmm.LogZero
			bestPrev := 0

			for prevS, prevPos := range model.POSTags {
				if dp[t-1][prevS] <= hmm.LogZero {
					continue
				}
				logTrans := model.LogTransition(prevPos, pos)
				if logTrans <= hmm.LogZero {
					continue
				}
				candidate := dp[t-1][prevS] + logTrans + logEmit
				if candidate > bestScore {
					bestScore = candidate
					bestPrev = prevS
				}
			}

			// If no valid transition found, use smoothed score with uniform transition
			if bestScore <= hmm.LogZero {
				for prevS := range model.POSTags {
					if dp[t-1][prevS] > hmm.LogZero {
						candidate := dp[t-1][prevS] + logEmit
						if candidate > bestScore {
							bestScore = candidate
							bestPrev = prevS
						}
					}
				}
			}

			dp[t][s] = bestScore
			bp[t][s] = bestPrev
		}
	}

	// Find best final state
	bestFinalScore := hmm.LogZero
	bestFinalState := 0
	for s := range model.POSTags {
		if dp[T-1][s] > bestFinalScore {
			bestFinalScore = dp[T-1][s]
			bestFinalState = s
		}
	}

	// Backtrack
	path := make([]int, T)
	path[T-1] = bestFinalState
	for t := T - 1; t > 0; t-- {
		path[t-1] = bp[t][path[t]]
	}

	// Build results
	results := make([]Result, T)
	for t, tok := range tokens {
		stateIdx := path[t]
		pos := model.POSTags[stateIdx]
		if hmm.IsValidPOS(tok.Tag) {
			pos = tok.Tag
		}
		results[t] = Result{
			Surface: tok.Surface,
			POS:     pos,
			Score:   dp[t][stateIdx],
		}
	}

	return results
}
