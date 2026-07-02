package morpho

import (
	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/dictionary"
	"github.com/7thCode/morpho/internal/hmm"
	"github.com/7thCode/morpho/internal/tokenizer"
	"github.com/7thCode/morpho/internal/viterbi"
)

// Morpheme represents a single morpheme with its surface form, reading, and POS information.
type Morpheme struct {
	Surface   string `json:"surface"`
	Reading   string `json:"reading,omitempty"`
	POS       string `json:"pos"`
	POSDetail string `json:"pos_detail,omitempty"`
}

// Analyzer performs Japanese morphological analysis.
type Analyzer struct {
	dictPath   string
	dictionary *dictionary.Dictionary
	trainer    *hmm.Trainer
}

// New creates an Analyzer, loading the dictionary from dictPath.
// If the file does not exist a fresh empty dictionary is used.
func New(dictPath string) (*Analyzer, error) {
	dict, err := dictionary.Load(dictPath)
	if err != nil {
		return nil, err
	}
	return &Analyzer{
		dictPath:   dictPath,
		dictionary: dict,
		trainer:    hmm.NewTrainer(),
	}, nil
}

// Train trains the HMM model from the given corpus text and updates the dictionary.
func (a *Analyzer) Train(corpus string) error {
	tokens := tokenizer.Segment(corpus)

	hmm.TrainOnTokens(tokens, a.trainer)
	a.dictionary.Model = a.trainer.Build()

	// Also update word entries in the dictionary
	for _, tok := range nonSpaceTokens(tokens) {
		if pos := hmm.InferPOS(tok); pos != "" {
			a.dictionary.Update(tok.Surface, pos)
		}
	}
	return nil
}

// Analyze performs morphological analysis on the input text.
// If no trained model is available it falls back to heuristic POS inference.
func (a *Analyzer) Analyze(text string) ([]Morpheme, error) {
	tokens := nonSpaceTokens(tokenizer.Segment(text))
	if len(tokens) == 0 {
		return nil, nil
	}

	model := a.dictionary.Model
	if model == nil || len(model.POSTags) == 0 {
		// Fallback: heuristic-only analysis
		morphemes := make([]Morpheme, len(tokens))
		for i, tok := range tokens {
			morphemes[i] = Morpheme{
				Surface: tok.Surface,
				POS:     hmm.InferPOS(tok),
			}
		}
		return morphemes, nil
	}

	results := viterbi.Decode(tokens, model)
	morphemes := make([]Morpheme, len(results))
	for i, r := range results {
		morphemes[i] = Morpheme{
			Surface: r.Surface,
			POS:     r.POS,
		}
	}
	return morphemes, nil
}

// Save persists the current dictionary (and model) to the given path.
func (a *Analyzer) Save(path string) error {
	return a.dictionary.Save(path)
}

// nonSpaceTokens filters out space tokens.
func nonSpaceTokens(tokens []tokenizer.Token) []tokenizer.Token {
	filtered := make([]tokenizer.Token, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Type != chartype.Space {
			filtered = append(filtered, tok)
		}
	}
	return filtered
}
