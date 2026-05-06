package hmm

import (
	"math"
	"strings"

	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/tokenizer"
)

// POS tag constants for Japanese.
const (
	POSNoun     = "名詞"
	POSVerb     = "動詞"
	POSAdj      = "形容詞"
	POSParticle = "助詞"
	POSAuxVerb  = "助動詞"
	POSAdverb   = "副詞"
	POSSymbol   = "記号"
	POSNumber   = "数詞"
	POSForeign  = "外来語"
	POSUnknown  = "未知語"
)

// particleSet contains common Japanese particles.
var particleSet = map[string]bool{
	"は": true, "が": true, "を": true, "に": true, "で": true,
	"と": true, "も": true, "の": true, "へ": true, "や": true,
	"か": true, "ね": true, "よ": true, "な": true, "から": true,
	"まで": true, "より": true, "ほど": true, "だけ": true, "しか": true,
	"ので": true, "のに": true, "って": true, "では": true, "には": true,
	"とは": true,
}

// auxVerbSet contains common Japanese auxiliary verbs.
var auxVerbSet = map[string]bool{
	"だ": true, "です": true, "ます": true, "た": true, "ない": true,
	"れる": true, "られる": true, "せる": true, "させる": true,
	"でした": true, "ました": true, "ません": true, "ないで": true,
}

// verbEndings contains verb-final characters in hiragana.
var verbEndings = map[rune]bool{
	'う': true, 'る': true, 'く': true, 'す': true,
	'ぬ': true, 'む': true, 'ぶ': true, 'つ': true, 'ぐ': true,
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
		total := 0.0
		for _, c := range targets {
			total += c
		}
		m.Transition[fromPos] = make(map[string]float64)
		for toPos, c := range targets {
			m.Transition[fromPos][toPos] = math.Log(c / total)
		}
	}

	// Normalize emission probabilities
	for pos, words := range t.emissionCounts {
		total := 0.0
		for _, c := range words {
			total += c
		}
		m.Emission[pos] = make(map[string]float64)
		for word, c := range words {
			m.Emission[pos][word] = math.Log(c / total)
		}
	}

	return m
}

// InferPOS heuristically infers the POS tag of a token based on its character type and content.
func InferPOS(token tokenizer.Token) string {
	switch token.Type {
	case chartype.Space:
		return ""
	case chartype.Symbol:
		return POSSymbol
	case chartype.Digit:
		return POSNumber
	case chartype.Latin:
		return POSForeign
	case chartype.Katakana:
		return POSForeign
	case chartype.Kanji:
		return POSNoun
	case chartype.Hiragana:
		return inferHiraganaPOS(token.Surface)
	default:
		// Mixed or unknown: check last rune
		return inferMixedPOS(token.Surface)
	}
}

// inferHiraganaPOS infers POS for a hiragana token.
func inferHiraganaPOS(surface string) string {
	if particleSet[surface] {
		return POSParticle
	}
	if auxVerbSet[surface] {
		return POSAuxVerb
	}
	runes := []rune(surface)
	if len(runes) == 0 {
		return POSUnknown
	}
	last := runes[len(runes)-1]
	if verbEndings[last] {
		return POSVerb
	}
	if last == 'い' || last == 'く' {
		return POSAdj
	}
	return POSAdverb
}

// inferMixedPOS infers POS for a mixed-type token based on its last rune.
func inferMixedPOS(surface string) string {
	runes := []rune(surface)
	if len(runes) == 0 {
		return POSUnknown
	}
	last := runes[len(runes)-1]
	if verbEndings[last] {
		return POSVerb
	}
	if last == 'い' {
		return POSAdj
	}
	return POSNoun
}

// TrainOnText segments text, groups into sentences, and trains the Trainer.
func TrainOnText(text string, trainer *Trainer) {
	tokens := tokenizer.Segment(text)
	if len(tokens) == 0 {
		return
	}

	sentenceEnders := map[rune]bool{
		'。': true, '！': true, '？': true, '\n': true,
		'!': true, '?': true,
	}

	var sentWords []string
	var sentPoses []string

	for _, tok := range tokens {
		// Check if this token is or ends with a sentence-ender
		if tok.Type == chartype.Space {
			continue
		}

		runes := []rune(tok.Surface)
		isSentenceEnd := false
		if len(runes) > 0 {
			lastRune := runes[len(runes)-1]
			if sentenceEnders[lastRune] {
				isSentenceEnd = true
			}
		}

		// Check if the entire token is a sentence-ending symbol
		if tok.Type == chartype.Symbol {
			allEnders := true
			for _, r := range runes {
				if !sentenceEnders[r] {
					allEnders = false
					break
				}
			}
			if allEnders {
				// Include the punctuation as 記号, then flush
				if len(sentWords) > 0 {
					sentWords = append(sentWords, tok.Surface)
					sentPoses = append(sentPoses, POSSymbol)
					trainer.AddSequence(sentWords, sentPoses)
					sentWords = nil
					sentPoses = nil
				}
				continue
			}
		}

		pos := InferPOS(tok)
		if pos == "" {
			continue
		}

		// Check if the surface itself is a newline/sentence ender
		trimmed := strings.TrimSpace(tok.Surface)
		if trimmed == "" {
			if len(sentWords) > 0 {
				trainer.AddSequence(sentWords, sentPoses)
				sentWords = nil
				sentPoses = nil
			}
			continue
		}

		sentWords = append(sentWords, tok.Surface)
		sentPoses = append(sentPoses, pos)

		if isSentenceEnd {
			trainer.AddSequence(sentWords, sentPoses)
			sentWords = nil
			sentPoses = nil
		}
	}

	// Flush remaining
	if len(sentWords) > 0 {
		trainer.AddSequence(sentWords, sentPoses)
	}
}
