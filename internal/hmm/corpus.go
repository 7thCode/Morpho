package hmm

import (
	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/tokenizer"
)

// sentenceEnders contains runes that terminate a sentence.
var sentenceEnders = map[rune]bool{
	'。': true, '！': true, '？': true, '\n': true,
	'!': true, '?': true,
}

// TrainOnText segments text, groups into sentences, and trains the Trainer.
func TrainOnText(text string, trainer *Trainer) {
	TrainOnTokens(tokenizer.Segment(text), trainer)
}

// TrainOnTokens groups pre-segmented tokens into sentences and trains the Trainer.
func TrainOnTokens(tokens []tokenizer.Token, trainer *Trainer) {
	var sentWords []string
	var sentPoses []string

	flush := func() {
		if len(sentWords) > 0 {
			trainer.AddSequence(sentWords, sentPoses)
			sentWords = nil
			sentPoses = nil
		}
	}

	for _, tok := range tokens {
		if tok.Type == chartype.Space {
			continue
		}

		runes := []rune(tok.Surface)
		isSentenceEnd := len(runes) > 0 && sentenceEnders[runes[len(runes)-1]]

		// A symbol token made entirely of sentence-enders closes the sentence:
		// include the punctuation as 記号, then flush.
		if tok.Type == chartype.Symbol && allSentenceEnders(runes) {
			if len(sentWords) > 0 {
				sentWords = append(sentWords, tok.Surface)
				sentPoses = append(sentPoses, POSSymbol)
				flush()
			}
			continue
		}

		pos := InferPOS(tok)
		if pos == "" {
			continue
		}

		sentWords = append(sentWords, tok.Surface)
		sentPoses = append(sentPoses, pos)

		if isSentenceEnd {
			flush()
		}
	}

	flush()
}

// allSentenceEnders reports whether every rune is a sentence-ending symbol.
func allSentenceEnders(runes []rune) bool {
	for _, r := range runes {
		if !sentenceEnders[r] {
			return false
		}
	}
	return true
}
