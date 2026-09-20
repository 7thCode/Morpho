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

var names = [...]string{"Hiragana", "Katakana", "Kanji", "Latin", "Digit", "Symbol", "Space"}

// String returns the type's name (e.g. "Kanji").
func (t CharType) String() string {
	if t < 0 || int(t) >= len(names) {
		return "Unknown"
	}
	return names[t]
}

// Of returns the CharType for the given rune.
func Of(r rune) CharType {
	switch {
	case unicode.IsSpace(r):
		return Space
	case r >= 0x3040 && r <= 0x309F:
		return Hiragana
	case (r >= 0x30A0 && r <= 0x30FF) || (r >= 0x31F0 && r <= 0x31FF) || (r >= 0xFF65 && r <= 0xFF9F):
		return Katakana
	case isKanji(r):
		return Kanji
	case (r >= '0' && r <= '9') || (r >= '０' && r <= '９'):
		return Digit
	case unicode.Is(unicode.Latin, r):
		return Latin
	default:
		return Symbol
	}
}

// isKanji covers the CJK ideograph blocks, compatibility ideographs, and
// the ideographic iteration/closing marks (々 〆 〇) that behave as part of
// a kanji word (時々, 人々) rather than as punctuation.
func isKanji(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs
		return true
	case r >= 0x3400 && r <= 0x4DBF: // Extension A
		return true
	case r >= 0x20000 && r <= 0x323AF: // Extensions B-G, compatibility supplement
		return true
	case r >= 0xF900 && r <= 0xFAFF: // Compatibility Ideographs (e.g. 﨑)
		return true
	case r >= 0x3005 && r <= 0x3007: // 々 〆 〇
		return true
	}
	return false
}
