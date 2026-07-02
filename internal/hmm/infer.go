package hmm

import (
	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/tokenizer"
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
