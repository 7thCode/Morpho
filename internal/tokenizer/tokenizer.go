package tokenizer

import "github.com/7thCode/morpho/internal/chartype"

// Token represents a segmented unit of text with its character type and position.
type Token struct {
	Surface string
	Type    chartype.CharType
	// StartPos and EndPos are rune offsets into the original text
	// (Surface == string(runes[StartPos:EndPos])).
	StartPos int
	EndPos   int
	// Tag is a label assigned before analysis (e.g. the POS from a
	// user-registered word). Empty for tokens found only by character type.
	Tag string
}

// Segment splits text on character-type boundaries and returns a slice of Tokens.
func Segment(text string) []Token {
	return SegmentWithLexicon(text, nil)
}

// SegmentWithLexicon splits text like Segment, but words registered in lex
// take priority: at each position the longest lexicon word starting there is
// emitted as its own Tag-ed token (even across character-type boundaries,
// e.g. 東京タワー), and the text in between is split by character type.
// A nil lexicon behaves exactly like Segment.
func SegmentWithLexicon(text string, lex *Lexicon) []Token {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var tokens []Token
	pending := 0
	for i := 0; i < len(runes); {
		if n, tag := lex.longestAt(runes, i); n > 0 {
			tokens = appendByCharType(tokens, runes, pending, i)
			tokens = append(tokens, Token{
				Surface:  string(runes[i : i+n]),
				Type:     chartype.Of(runes[i]),
				StartPos: i,
				EndPos:   i + n,
				Tag:      tag,
			})
			i += n
			pending = i
			continue
		}
		i++
	}
	return appendByCharType(tokens, runes, pending, len(runes))
}

// appendByCharType splits runes[from:to] at character-type boundaries.
func appendByCharType(tokens []Token, runes []rune, from, to int) []Token {
	if from >= to {
		return tokens
	}
	start := from
	cur := chartype.Of(runes[from])
	for i := from + 1; i < to; i++ {
		if t := chartype.Of(runes[i]); t != cur && !continuesRun(cur, runes[i]) {
			tokens = append(tokens, Token{Surface: string(runes[start:i]), Type: cur, StartPos: start, EndPos: i})
			start = i
			cur = t
		}
	}
	return append(tokens, Token{Surface: string(runes[start:to]), Type: cur, StartPos: start, EndPos: to})
}

// continuesRun reports whether r, although classified as a different type,
// belongs to the current run. The prolonged sound mark ー is classified as
// Katakana but is routinely written inside hiragana words (らーめん).
func continuesRun(cur chartype.CharType, r rune) bool {
	return cur == chartype.Hiragana && r == 'ー'
}

// Lexicon is a set of words (with an optional tag each) that SegmentWithLexicon
// keeps intact. The zero value and nil are both an empty lexicon.
type Lexicon struct {
	words  map[string]string
	starts map[rune]struct{}
	maxLen int
}

// NewLexicon builds a Lexicon from surface->tag pairs. Empty surfaces and
// surfaces containing whitespace are ignored since they can never match a
// single non-space run of text.
func NewLexicon(words map[string]string) *Lexicon {
	l := &Lexicon{words: map[string]string{}, starts: map[rune]struct{}{}}
	for surface, tag := range words {
		runes := []rune(surface)
		if len(runes) == 0 || hasSpace(runes) {
			continue
		}
		l.words[surface] = tag
		l.starts[runes[0]] = struct{}{}
		if len(runes) > l.maxLen {
			l.maxLen = len(runes)
		}
	}
	return l
}

// Len returns the number of words in the lexicon.
func (l *Lexicon) Len() int {
	if l == nil {
		return 0
	}
	return len(l.words)
}

// longestAt returns the rune length and tag of the longest lexicon word that
// starts at runes[i], or 0 if none does.
func (l *Lexicon) longestAt(runes []rune, i int) (int, string) {
	if l == nil || len(l.words) == 0 {
		return 0, ""
	}
	if _, ok := l.starts[runes[i]]; !ok {
		return 0, ""
	}
	for n := min(l.maxLen, len(runes)-i); n >= 1; n-- {
		if tag, ok := l.words[string(runes[i:i+n])]; ok {
			return n, tag
		}
	}
	return 0, ""
}

func hasSpace(runes []rune) bool {
	for _, r := range runes {
		if chartype.Of(r) == chartype.Space {
			return true
		}
	}
	return false
}
