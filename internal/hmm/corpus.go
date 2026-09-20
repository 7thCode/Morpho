package hmm

import (
	"strings"

	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/tokenizer"
)

// sentenceEnders contains runes that terminate a sentence.
var sentenceEnders = map[rune]bool{
	'。': true, '！': true, '？': true,
	'!': true, '?': true,
}

// SplitSentences groups tokens into sentences. Space tokens are dropped, but
// a Space token containing a line break ends the current sentence, and a
// Symbol token containing a sentence-ender (。！？!?) ends it after that token.
// Training and analysis both use this, so the two see the same sentences.
func SplitSentences(tokens []tokenizer.Token) [][]tokenizer.Token {
	var sentences [][]tokenizer.Token
	var cur []tokenizer.Token
	flush := func() {
		if len(cur) > 0 {
			sentences = append(sentences, cur)
			cur = nil
		}
	}

	for _, tok := range tokens {
		switch {
		case tok.Type == chartype.Space:
			if strings.ContainsAny(tok.Surface, "\r\n") {
				flush()
			}
		case tok.Type == chartype.Symbol && strings.ContainsAny(tok.Surface, "。！？!?"):
			cur = append(cur, tok)
			flush()
		default:
			cur = append(cur, tok)
		}
	}
	flush()
	return sentences
}

// Sentence is a labelled token sequence: Words[i] is tagged POS[i].
type Sentence struct {
	Words []string
	POS   []string
}

// TrainOnText adds every sentence of text to trainer and returns how many
// sentences it contributed. See ParseCorpus for the accepted formats.
func TrainOnText(text string, trainer *Trainer) int {
	return TrainOnTextWithLexicon(text, nil, trainer)
}

// TrainOnTextWithLexicon is TrainOnText with a lexicon guiding segmentation
// and labelling of plain lines.
func TrainOnTextWithLexicon(text string, lex *tokenizer.Lexicon, trainer *Trainer) int {
	sentences := ParseCorpus(text, lex)
	for _, s := range sentences {
		trainer.AddSequence(s.Words, s.POS)
	}
	return len(sentences)
}

// ParseCorpus turns a corpus into labelled sentences, one line at a time.
//
// A line whose whitespace-separated fields all look like "word/POS" with a
// known POS tag is treated as hand-labelled and used exactly as written. Any
// other line is segmented by character type (honouring lex, if given) and
// labelled heuristically: a lexicon word carries its registered POS,
// everything else gets InferPOS. Sentences containing nothing but symbols
// carry no grammar and are dropped.
func ParseCorpus(text string, lex *tokenizer.Lexicon) []Sentence {
	var out []Sentence
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if words, poses, ok := parseAnnotatedLine(line); ok {
			out = appendLabelled(out, words, poses)
			continue
		}
		for _, sentence := range SplitSentences(tokenizer.SegmentWithLexicon(line, lex)) {
			words := make([]string, len(sentence))
			poses := make([]string, len(sentence))
			for i, tok := range sentence {
				words[i] = tok.Surface
				poses[i] = LabelOf(tok)
			}
			out = appendLabelled(out, words, poses)
		}
	}
	return out
}

// LabelOf returns the POS a token should be trained with: its registered
// tag if it has a valid one, otherwise the character-type heuristic.
func LabelOf(tok tokenizer.Token) string {
	if IsValidPOS(tok.Tag) {
		return tok.Tag
	}
	return InferPOS(tok)
}

// appendLabelled splits a labelled sequence at sentence-ending symbols and
// appends the pieces that contain at least one non-symbol.
func appendLabelled(out []Sentence, words, poses []string) []Sentence {
	start := 0
	emit := func(end int) {
		if end <= start {
			return
		}
		w, p := words[start:end], poses[start:end]
		start = end
		for _, tag := range p {
			if tag != POSSymbol {
				out = append(out, Sentence{Words: w, POS: p})
				return
			}
		}
	}
	for i, w := range words {
		if poses[i] == POSSymbol && strings.ContainsAny(w, "。！？!?") {
			emit(i + 1)
		}
	}
	emit(len(words))
	return out
}

// parseAnnotatedLine parses "word/POS word/POS ..." lines. The POS is what
// follows the last "/" of each field, so words may themselves contain "/".
func parseAnnotatedLine(line string) (words, poses []string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, false
	}
	for _, f := range fields {
		i := strings.LastIndex(f, "/")
		if i <= 0 || !IsValidPOS(f[i+1:]) {
			return nil, nil, false
		}
		words = append(words, f[:i])
		poses = append(poses, f[i+1:])
	}
	return words, poses, true
}
