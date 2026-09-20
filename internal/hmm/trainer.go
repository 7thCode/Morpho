package hmm

import (
	"math"
	"sort"

	"github.com/7thCode/morpho/internal/chartype"
)

// smoothing is the Lidstone (add-alpha) constant applied to the initial and
// transition distributions. It is deliberately small so that observed
// patterns dominate, but non-zero so no POS sequence is impossible.
const smoothing = 0.1

// Counts is the serializable form of a Trainer's accumulated counts.
type Counts struct {
	Initial    map[string]float64            `json:"initial,omitempty"`
	Transition map[string]map[string]float64 `json:"transition,omitempty"`
	Emission   map[string]map[string]float64 `json:"emission,omitempty"`
}

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

// NewTrainerFromCounts returns a Trainer seeded with a deep copy of c, so
// that training can resume where a previous run left off.
func NewTrainerFromCounts(c Counts) *Trainer {
	t := NewTrainer()
	for pos, n := range c.Initial {
		t.initialCounts[pos] = n
	}
	t.transitionCounts = copyNested(c.Transition)
	t.emissionCounts = copyNested(c.Emission)
	return t
}

// Snapshot returns a deep copy of the accumulated counts.
func (t *Trainer) Snapshot() Counts {
	initial := make(map[string]float64, len(t.initialCounts))
	for pos, n := range t.initialCounts {
		initial[pos] = n
	}
	return Counts{
		Initial:    initial,
		Transition: copyNested(t.transitionCounts),
		Emission:   copyNested(t.emissionCounts),
	}
}

func copyNested(src map[string]map[string]float64) map[string]map[string]float64 {
	dst := make(map[string]map[string]float64, len(src))
	for k, inner := range src {
		c := make(map[string]float64, len(inner))
		for k2, v := range inner {
			c[k2] = v
		}
		dst[k] = c
	}
	return dst
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

// Build turns the counts into a Model.
//
//   - Initial and Transition are Lidstone-smoothed over every observed POS,
//     so unseen POS pairs are improbable rather than impossible.
//   - Emission uses Witten-Bell discounting: seen words get c/(N+T), and the
//     remaining T/(N+T) is reserved for unseen words (UnknownMass), divided
//     among character classes by ClassEmission.
//
// POSTags is sorted so the model, and the JSON it is saved to, is
// deterministic.
func (t *Trainer) Build() *Model {
	m := New()

	posSet := make(map[string]bool)
	for pos := range t.initialCounts {
		posSet[pos] = true
	}
	for pos, targets := range t.transitionCounts {
		posSet[pos] = true
		for to := range targets {
			posSet[to] = true
		}
	}
	for pos := range t.emissionCounts {
		posSet[pos] = true
	}
	for pos := range posSet {
		m.POSTags = append(m.POSTags, pos)
	}
	sort.Strings(m.POSTags)
	if len(m.POSTags) == 0 {
		return m
	}
	states := float64(len(m.POSTags))

	initTotal := 0.0
	for _, c := range t.initialCounts {
		initTotal += c
	}
	for _, pos := range m.POSTags {
		m.Initial[pos] = math.Log((t.initialCounts[pos] + smoothing) / (initTotal + smoothing*states))
	}

	for _, from := range m.POSTags {
		row := t.transitionCounts[from]
		total := 0.0
		for _, c := range row {
			total += c
		}
		m.Transition[from] = make(map[string]float64, len(m.POSTags))
		for _, to := range m.POSTags {
			m.Transition[from][to] = math.Log((row[to] + smoothing) / (total + smoothing*states))
		}
	}

	m.UnknownMass = make(map[string]float64, len(t.emissionCounts))
	m.ClassEmission = make(map[string]map[string]float64, len(t.emissionCounts))
	for pos, words := range t.emissionCounts {
		tokens := 0.0
		classTypes := make(map[string]float64)
		for w, c := range words {
			tokens += c
			classTypes[ClassOf(w)]++
		}
		types := float64(len(words))

		emission := make(map[string]float64, len(words))
		for w, c := range words {
			emission[w] = math.Log(c / (tokens + types))
		}
		m.Emission[pos] = emission
		m.UnknownMass[pos] = math.Log(types / (tokens + types))

		classes := make(map[string]float64)
		for c := chartype.Hiragana; c <= chartype.Space; c++ {
			classes[c.String()] = math.Log((classTypes[c.String()] + smoothing) / (types + smoothing*float64(chartype.Space+1)))
		}
		m.ClassEmission[pos] = classes
	}

	m.Prepare()
	return m
}
