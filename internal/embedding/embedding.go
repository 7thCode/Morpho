// Package embedding trains word vectors from tokenized sentences using
// skip-gram with negative sampling (Mikolov et al., 2013), so that words
// segmented by Morpho's own tokenizer can be embedded without depending on
// a pretrained model whose vocabulary was built with a different segmenter.
package embedding

import (
	"math"
	"math/rand"
	"sort"
)

// Model holds trained word vectors indexed by surface form.
type Model struct {
	Dim     int
	Vectors map[string][]float64
}

// Config controls skip-gram-with-negative-sampling training.
type Config struct {
	Dim       int     // vector dimensionality
	Window    int     // context window size on each side of the center word
	Negatives int     // negative samples drawn per positive pair
	Epochs    int     // passes over the corpus
	LearnRate float64 // initial learning rate, linearly decayed toward zero
	MinCount  int     // minimum word frequency to enter the vocabulary
	Seed      int64   // RNG seed, for reproducible training
}

// DefaultConfig returns settings tuned for small proof-of-concept corpora.
func DefaultConfig() Config {
	return Config{
		Dim:       50,
		Window:    2,
		Negatives: 5,
		Epochs:    50,
		LearnRate: 0.025,
		MinCount:  1,
		Seed:      1,
	}
}

const negativeTableSize = 100_000

// Train learns word vectors from tokenized sentences.
func Train(sentences [][]string, cfg Config) *Model {
	freq := make(map[string]int)
	for _, sent := range sentences {
		for _, w := range sent {
			freq[w]++
		}
	}

	vocab := make([]string, 0, len(freq))
	for w, c := range freq {
		if c >= cfg.MinCount {
			vocab = append(vocab, w)
		}
	}
	sort.Strings(vocab) // deterministic initialization order

	if len(vocab) == 0 {
		return &Model{Dim: cfg.Dim, Vectors: map[string][]float64{}}
	}

	rng := rand.New(rand.NewSource(cfg.Seed))

	syn0 := make(map[string][]float64, len(vocab))
	syn1 := make(map[string][]float64, len(vocab))
	for _, w := range vocab {
		syn0[w] = randomVector(rng, cfg.Dim)
		syn1[w] = make([]float64, cfg.Dim)
	}

	negTable := buildNegativeTable(vocab, freq)

	totalPairs := 0
	for _, sent := range sentences {
		totalPairs += len(sent)
	}
	if totalPairs == 0 {
		totalPairs = 1
	}
	totalSteps := cfg.Epochs * totalPairs

	steps := 0
	for range cfg.Epochs {
		for _, sent := range sentences {
			filtered := onlyInVocab(sent, syn0)
			for i, center := range filtered {
				lr := learningRate(cfg.LearnRate, steps, totalSteps)
				start := max(i-cfg.Window, 0)
				end := min(i+cfg.Window, len(filtered)-1)
				for j := start; j <= end; j++ {
					if j == i {
						continue
					}
					trainPair(syn0[center], syn1, filtered[j], negTable, cfg.Negatives, lr, rng)
				}
				steps++
			}
		}
	}

	return &Model{Dim: cfg.Dim, Vectors: syn0}
}

// trainPair applies one skip-gram-with-negative-sampling update for the
// (center, context) pair, mutating centerVec and the relevant syn1 rows.
func trainPair(centerVec []float64, syn1 map[string][]float64, context string, negTable []string, negatives int, lr float64, rng *rand.Rand) {
	dim := len(centerVec)
	neu1e := make([]float64, dim)

	update := func(target string, label float64) {
		vec, ok := syn1[target]
		if !ok {
			return
		}
		pred := sigmoid(dotProduct(centerVec, vec))
		grad := (label - pred) * lr
		for d := range dim {
			neu1e[d] += grad * vec[d]
			vec[d] += grad * centerVec[d]
		}
	}

	update(context, 1)
	for range negatives {
		neg := negTable[rng.Intn(len(negTable))]
		if neg == context {
			continue
		}
		update(neg, 0)
	}

	for d := range dim {
		centerVec[d] += neu1e[d]
	}
}

// buildNegativeTable builds a sampling table weighted by freq^0.75, the
// smoothing exponent used by the original word2vec implementation.
func buildNegativeTable(vocab []string, freq map[string]int) []string {
	pow := make([]float64, len(vocab))
	var totalPow float64
	for i, w := range vocab {
		p := math.Pow(float64(freq[w]), 0.75)
		pow[i] = p
		totalPow += p
	}

	table := make([]string, 0, negativeTableSize)
	i := 0
	cumulative := pow[0] / totalPow
	for a := range negativeTableSize {
		table = append(table, vocab[i])
		if float64(a)/float64(negativeTableSize) > cumulative && i < len(vocab)-1 {
			i++
			cumulative += pow[i] / totalPow
		}
	}
	return table
}

func onlyInVocab(sent []string, vocab map[string][]float64) []string {
	out := make([]string, 0, len(sent))
	for _, w := range sent {
		if _, ok := vocab[w]; ok {
			out = append(out, w)
		}
	}
	return out
}

func learningRate(initial float64, step, totalSteps int) float64 {
	if totalSteps <= 0 {
		return initial
	}
	rate := initial * (1 - float64(step)/float64(totalSteps))
	if min := initial * 0.0001; rate < min {
		rate = min
	}
	return rate
}

func randomVector(rng *rand.Rand, dim int) []float64 {
	v := make([]float64, dim)
	for i := range v {
		v[i] = (rng.Float64() - 0.5) / float64(dim)
	}
	return v
}

func sigmoid(x float64) float64 {
	switch {
	case x > 6:
		return 1
	case x < -6:
		return 0
	default:
		return 1 / (1 + math.Exp(-x))
	}
}

func dotProduct(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// Similarity returns the cosine similarity between two vectors of equal length.
func Similarity(a, b []float64) float64 {
	na := math.Sqrt(dotProduct(a, a))
	nb := math.Sqrt(dotProduct(b, b))
	if na == 0 || nb == 0 {
		return 0
	}
	return dotProduct(a, b) / (na * nb)
}

// Neighbor pairs a word with its similarity score to a query word.
type Neighbor struct {
	Word  string
	Score float64
}

// Nearest returns the n words most similar to word by cosine similarity,
// excluding word itself. It returns nil if word is not in the model.
func (m *Model) Nearest(word string, n int) []Neighbor {
	target, ok := m.Vectors[word]
	if !ok {
		return nil
	}
	neighbors := make([]Neighbor, 0, len(m.Vectors))
	for w, v := range m.Vectors {
		if w == word {
			continue
		}
		neighbors = append(neighbors, Neighbor{Word: w, Score: Similarity(target, v)})
	}
	sort.Slice(neighbors, func(i, j int) bool {
		if neighbors[i].Score != neighbors[j].Score {
			return neighbors[i].Score > neighbors[j].Score
		}
		return neighbors[i].Word < neighbors[j].Word
	})
	if n < len(neighbors) {
		neighbors = neighbors[:n]
	}
	return neighbors
}
