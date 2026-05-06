package chartype

import "unicode"

// CharType represents the type of a Japanese or general character.
type CharType int

const (
	Hiragana CharType = iota
	Katakana
	Kanji
	Latin
	Digit
	Symbol
	Space
)

// Of returns the CharType for the given rune.
func Of(r rune) CharType {
	switch {
	case unicode.IsSpace(r):
		return Space
	case r >= '぀' && r <= 'ゟ':
		return Hiragana
	case (r >= '゠' && r <= 'ヿ') || (r >= '･' && r <= 'ﾟ'):
		return Katakana
	case (r >= '一' && r <= '鿿') || (r >= '㐀' && r <= '䶿'):
		return Kanji
	case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= 'Ａ' && r <= 'ｚ'):
		return Latin
	case (r >= '0' && r <= '9') || (r >= '０' && r <= '９'):
		return Digit
	default:
		return Symbol
	}
}
