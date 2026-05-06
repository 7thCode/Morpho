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
			bp[t][s] = -1
		}
	}

	// Initialize: t=0
	for s, pos := range model.POSTags {
		logInit := model.LogInitial(pos)
		logEmit := model.SmoothEmission(pos, tokens[0].Surface)
		if logInit > hmm.LogZero {
			dp[0][s] = logInit + logEmit
		} else {
			dp[0][s] = hmm.LogZero
		}
	}

	// Recursion
	for t := 1; t < T; t++ {
		for s, pos := range model.POSTags {
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
		results[t] = Result{
			Surface: tok.Surface,
			POS:     model.POSTags[stateIdx],
			Score:   dp[t][stateIdx],
		}
	}

	return results
}
