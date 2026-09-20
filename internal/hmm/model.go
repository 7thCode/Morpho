package hmm

import (
	"math"

	"github.com/7thCode/morpho/internal/chartype"
)

// LogZero represents negative infinity in log-probability space.
const LogZero = -1e300

// Model holds the HMM parameters for part-of-speech tagging.
//
// Models built by Trainer.Build have dense, smoothed Initial/Transition
// tables (every POS pair has a probability) and a Witten-Bell style
// unknown-word model (UnknownMass + ClassEmission). Models loaded from older
// dictionary files lack the unknown-word fields; SmoothEmission then falls
// back to the original add-one estimate.
type Model struct {
	Initial    map[string]float64            `json:"initial"`
	Transition map[string]map[string]float64 `json:"transition"`
	Emission   map[string]map[string]float64 `json:"emission"`
	POSTags    []string                      `json:"pos_tags"`

	// UnknownMass[pos] is log P(next word is unseen | pos).
	UnknownMass map[string]float64 `json:"unknown_mass,omitempty"`
	// ClassEmission[pos][charclass] is log P(character class | pos, unseen
	// word), estimated over distinct words so that frequent words do not
	// drown out the shape of the open vocabulary.
	ClassEmission map[string]map[string]float64 `json:"class_emission,omitempty"`

	// vocabCount caches vocabSize; 0 means not yet computed. It is filled by
	// Prepare so that concurrent readers never write to the model.
	vocabCount int
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

// Prepare computes derived, cached values. Call it once after building or
// loading a model and before sharing it between goroutines.
func (m *Model) Prepare() {
	m.vocabCount = m.countVocab()
}

// HasPOS reports whether pos is a state of the model.
func (m *Model) HasPOS(pos string) bool {
	for _, p := range m.POSTags {
		if p == pos {
			return true
		}
	}
	return false
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

// SmoothEmission returns an emission log-probability that is defined for
// every (pos, word) pair. Seen words return LogEmission. Unseen words get the
// POS's unknown-word mass split by the word's character class, so that e.g.
// an unseen number is far likelier under 数詞 than under 助詞.
func (m *Model) SmoothEmission(pos, word string) float64 {
	if inner, ok := m.Emission[pos]; ok {
		if v, ok := inner[word]; ok {
			return v
		}
	}

	if len(m.ClassEmission) == 0 {
		return m.legacySmoothEmission(pos, word)
	}

	floor := math.Log(1.0 / float64(m.vocabSize()+1))
	mass, ok := m.UnknownMass[pos]
	if !ok {
		return floor
	}
	class, ok := m.ClassEmission[pos][ClassOf(word)]
	if !ok {
		return floor + mass
	}
	// The unseen mass is spread over an unseen-word space approximated by the
	// vocabulary size; the constant cancels out when comparing POS tags.
	return mass + class - math.Log(float64(m.vocabSize()+1))
}

// legacySmoothEmission is the add-one estimate used by models that predate
// the unknown-word model.
func (m *Model) legacySmoothEmission(pos, word string) float64 {
	if inner, ok := m.Emission[pos]; ok {
		return math.Log(1.0 / (float64(len(inner)) + float64(m.vocabSize()) + 1.0))
	}
	return math.Log(1.0 / (float64(m.vocabSize()) + 1.0))
}

// vocabSize returns the number of distinct words across all POS emissions.
// It uses the value cached by Prepare; without it the count is recomputed on
// every call (correct, just slower) rather than cached, so unprepared models
// stay safe for concurrent readers.
func (m *Model) vocabSize() int {
	if m.vocabCount > 0 {
		return m.vocabCount
	}
	return m.countVocab()
}

func (m *Model) countVocab() int {
	vocab := make(map[string]struct{})
	for _, words := range m.Emission {
		for w := range words {
			vocab[w] = struct{}{}
		}
	}
	return len(vocab)
}

// ClassOf returns the character-class name of a word, judged by its first
// rune. It keys Model.ClassEmission.
func ClassOf(word string) string {
	for _, r := range word {
		return chartype.Of(r).String()
	}
	return chartype.Space.String()
}
