package hmm

import "math"

// Trainer accumulates counts for building an HMM model.
type Trainer struct {
	initialCounts    map[string]float64
	transitionCounts map[string]map[string]float64
	emissionCounts   map[string]map[string]float64
}

// NewTrainer returns a new, empty Trainer.
func NewTrainer() *Trainer {
	return &Trainer{
		initialCounts:    make(map[string]float64),
		transitionCounts: make(map[string]map[string]float64),
		emissionCounts:   make(map[string]map[string]float64),
	}
}

// AddSequence updates counts from a parallel slice of words and POS tags.
func (t *Trainer) AddSequence(words, poses []string) {
	if len(words) == 0 || len(words) != len(poses) {
		return
	}

	// Initial counts
	t.initialCounts[poses[0]]++

	for i, pos := range poses {
		word := words[i]
		// Emission counts
		if _, ok := t.emissionCounts[pos]; !ok {
			t.emissionCounts[pos] = make(map[string]float64)
		}
		t.emissionCounts[pos][word]++

		// Transition counts
		if i < len(poses)-1 {
			nextPos := poses[i+1]
			if _, ok := t.transitionCounts[pos]; !ok {
				t.transitionCounts[pos] = make(map[string]float64)
			}
			t.transitionCounts[pos][nextPos]++
		}
	}
}

// Build normalizes counts into log-probabilities and returns a Model.
func (t *Trainer) Build() *Model {
	m := New()

	// Collect all POS tags
	posSet := make(map[string]bool)
	for pos := range t.initialCounts {
		posSet[pos] = true
	}
	for pos := range t.transitionCounts {
		posSet[pos] = true
	}
	for pos := range t.emissionCounts {
		posSet[pos] = true
	}

	for pos := range posSet {
		m.POSTags = append(m.POSTags, pos)
	}

	// Normalize initial probabilities
	initTotal := 0.0
	for _, c := range t.initialCounts {
		initTotal += c
	}
	for pos, c := range t.initialCounts {
		m.Initial[pos] = math.Log(c / initTotal)
	}

	// Normalize transition probabilities
	for fromPos, targets := range t.transitionCounts {
		m.Transition[fromPos] = normalizeLog(targets)
	}

	// Normalize emission probabilities
	for pos, words := range t.emissionCounts {
		m.Emission[pos] = normalizeLog(words)
	}

	return m
}

// normalizeLog converts raw counts into log-probabilities.
func normalizeLog(counts map[string]float64) map[string]float64 {
	total := 0.0
	for _, c := range counts {
		total += c
	}
	probs := make(map[string]float64, len(counts))
	for key, c := range counts {
		probs[key] = math.Log(c / total)
	}
	return probs
}
