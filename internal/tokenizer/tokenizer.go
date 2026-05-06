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
	if text == "" {
		return nil
	}

	var tokens []Token
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	startPos := 0
	currentType := chartype.Of(runes[0])
	start := 0

	for i := 1; i < len(runes); i++ {
		t := chartype.Of(runes[i])
		if t != currentType {
			surface := string(runes[start:i])
			tokens = append(tokens, Token{
				Surface:  surface,
				Type:     currentType,
				StartPos: startPos,
				EndPos:   startPos + len(surface),
			})
			startPos += len(surface)
			start = i
			currentType = t
		}
	}

	// Append the last token
	surface := string(runes[start:])
	tokens = append(tokens, Token{
		Surface:  surface,
		Type:     currentType,
		StartPos: startPos,
		EndPos:   startPos + len(surface),
	})

	return tokens
}
