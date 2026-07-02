package tokenizer

import "github.com/7thCode/morpho/internal/chartype"

// Token represents a segmented unit of text with its character type and position.
type Token struct {
	Surface  string
	Type     chartype.CharType
	StartPos int
	EndPos   int
}

// Segment splits text on character-type boundaries and returns a slice of Tokens.
func Segment(text string) []Token {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var tokens []Token
	appendToken := func(start, end int, t chartype.CharType) {
		surface := string(runes[start:end])
		startPos := 0
		if len(tokens) > 0 {
			startPos = tokens[len(tokens)-1].EndPos
		}
		tokens = append(tokens, Token{
			Surface:  surface,
			Type:     t,
			StartPos: startPos,
			EndPos:   startPos + len(surface),
		})
	}

	start := 0
	currentType := chartype.Of(runes[0])
	for i := 1; i < len(runes); i++ {
		if t := chartype.Of(runes[i]); t != currentType {
			appendToken(start, i, currentType)
			start = i
			currentType = t
		}
	}
	appendToken(start, len(runes), currentType)

	return tokens
}
