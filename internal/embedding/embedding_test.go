package embedding

import (
	"math"
	"testing"
)

func TestSimilarity(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{1, 0}
	c := []float64{0, 1}

	if got := Similarity(a, b); math.Abs(got-1) > 1e-9 {
		t.Errorf("Similarity(a, b) = %v, want 1", got)
	}
	if got := Similarity(a, c); math.Abs(got) > 1e-9 {
		t.Errorf("Similarity(a, c) = %v, want 0", got)
	}
}

func TestTrainProducesVectorsForVocab(t *testing.T) {
	sentences := [][]string{
		{"犬", "は", "動物", "だ"},
		{"猫", "は", "動物", "だ"},
	}
	cfg := DefaultConfig()
	cfg.Epochs = 5
	model := Train(sentences, cfg)

	for _, w := range []string{"犬", "猫", "動物", "は", "だ"} {
		v, ok := model.Vectors[w]
		if !ok {
			t.Fatalf("expected vector for %q", w)
		}
		if len(v) != cfg.Dim {
			t.Errorf("vector for %q has dim %d, want %d", w, len(v), cfg.Dim)
		}
		for _, x := range v {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				t.Fatalf("vector for %q contains NaN/Inf: %v", w, v)
			}
		}
	}
}

func TestTrainClustersCooccurringWords(t *testing.T) {
	// "犬" and "猫" always appear with "動物", while "パン" always appears
	// with "食べ物", so same-category words should end up more similar to
	// each other than to a word from the other category.
	sentences := [][]string{
		{"犬", "は", "動物", "だ"},
		{"猫", "は", "動物", "だ"},
		{"犬", "は", "動物", "だ"},
		{"猫", "は", "動物", "だ"},
		{"パン", "は", "食べ物", "だ"},
		{"パン", "は", "食べ物", "だ"},
	}
	cfg := DefaultConfig()
	cfg.Epochs = 100
	cfg.Seed = 42
	model := Train(sentences, cfg)

	simSameCategory := Similarity(model.Vectors["犬"], model.Vectors["猫"])
	simDiffCategory := Similarity(model.Vectors["犬"], model.Vectors["パン"])

	if simSameCategory <= simDiffCategory {
		t.Errorf("expected 犬-猫 similarity (%v) > 犬-パン similarity (%v)", simSameCategory, simDiffCategory)
	}
}

func TestNearestExcludesSelfAndRespectsN(t *testing.T) {
	sentences := [][]string{{"a", "b", "c", "d"}}
	cfg := DefaultConfig()
	cfg.Epochs = 5
	model := Train(sentences, cfg)

	neighbors := model.Nearest("a", 2)
	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}
	for _, n := range neighbors {
		if n.Word == "a" {
			t.Errorf("Nearest should not include the query word itself")
		}
	}
}

func TestNearestUnknownWord(t *testing.T) {
	sentences := [][]string{{"a", "b"}}
	model := Train(sentences, DefaultConfig())

	if got := model.Nearest("z", 3); got != nil {
		t.Errorf("Nearest for unknown word = %v, want nil", got)
	}
}
