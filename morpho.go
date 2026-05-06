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
	hmm.TrainOnText(corpus, a.trainer)
	model := a.trainer.Build()
	a.dictionary.Model = model

	// Also update word entries in the dictionary
	tokens := tokenizer.Segment(corpus)
	for _, tok := range tokens {
		if tok.Type != chartype.Space {
			pos := hmm.InferPOS(tok)
			if pos != "" {
				a.dictionary.Update(tok.Surface, pos)
			}
		}
	}
	return nil
}

// Analyze performs morphological analysis on the input text.
// If no trained model is available it falls back to heuristic POS inference.
func (a *Analyzer) Analyze(text string) ([]Morpheme, error) {
	tokens := tokenizer.Segment(text)

	if a.dictionary.Model == nil || len(a.dictionary.Model.POSTags) == 0 {
		// Fallback: heuristic-only analysis
		var result []Morpheme
		for _, tok := range tokens {
			if tok.Type == chartype.Space {
				continue
			}
			result = append(result, Morpheme{
				Surface: tok.Surface,
				POS:     hmm.InferPOS(tok),
			})
		}
		return result, nil
	}

	// Filter out space tokens for Viterbi
	var nonSpaceTokens []tokenizer.Token
	for _, tok := range tokens {
		if tok.Type != chartype.Space {
			nonSpaceTokens = append(nonSpaceTokens, tok)
		}
	}
	if len(nonSpaceTokens) == 0 {
		return nil, nil
	}

	vitResults := viterbi.Decode(nonSpaceTokens, a.dictionary.Model)
	var morphemes []Morpheme
	for _, r := range vitResults {
		morphemes = append(morphemes, Morpheme{
			Surface: r.Surface,
			POS:     r.POS,
		})
	}
	return morphemes, nil
}

// Save persists the current dictionary (and model) to the given path.
func (a *Analyzer) Save(path string) error {
	return a.dictionary.Save(path)
}
