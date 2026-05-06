package hmm

import "math"

// LogZero represents negative infinity in log-probability space.
const LogZero = -1e300

// Model holds the HMM parameters for part-of-speech tagging.
type Model struct {
	Initial    map[string]float64            `json:"initial"`
	Transition map[string]map[string]float64 `json:"transition"`
	Emission   map[string]map[string]float64 `json:"emission"`
	POSTags    []string                       `json:"pos_tags"`
}

// New creates and returns an empty HMM Model.
func New() *Model {
	return &Model{
		Initial:    make(map[string]float64),
		Transition: make(map[string]map[string]float64),
		Emission:   make(map[string]map[string]float64),
		POSTags:    []string{},
	}
}

// LogInitial returns the log-probability of pos being the initial state.
func (m *Model) LogInitial(pos string) float64 {
	if v, ok := m.Initial[pos]; ok {
		return v
	}
	return LogZero
}

// LogTransition returns the log-probability of transitioning from 'from' to 'to'.
func (m *Model) LogTransition(from, to string) float64 {
	if inner, ok := m.Transition[from]; ok {
		if v, ok := inner[to]; ok {
			return v
		}
	}
	return LogZero
}

// LogEmission returns the log-probability of emitting 'word' from state 'pos'.
func (m *Model) LogEmission(pos, word string) float64 {
	if inner, ok := m.Emission[pos]; ok {
		if v, ok := inner[word]; ok {
			return v
		}
	}
	return LogZero
}

// SmoothEmission returns a smoothed emission probability for unknown words.
// If the word is known, it returns LogEmission. Otherwise it uses add-one smoothing.
func (m *Model) SmoothEmission(pos, word string) float64 {
	if inner, ok := m.Emission[pos]; ok {
		if v, ok2 := inner[word]; ok2 {
			return v
		}
		// Unknown word: add-one (Laplace) smoothing
		totalEmissions := float64(len(inner))
		vocabSize := m.vocabSize()
		return math.Log(1.0 / (totalEmissions + float64(vocabSize) + 1.0))
	}
	// POS not seen at all
	vocabSize := m.vocabSize()
	return math.Log(1.0 / (float64(vocabSize) + 1.0))
}

// vocabSize returns the total number of distinct words across all POS emissions.
func (m *Model) vocabSize() int {
	vocab := make(map[string]struct{})
	for _, words := range m.Emission {
		for w := range words {
			vocab[w] = struct{}{}
		}
	}
	return len(vocab)
}
