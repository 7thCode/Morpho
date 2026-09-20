package chartype

import "testing"

func TestOf(t *testing.T) {
	tests := []struct {
		r    rune
		want CharType
		desc string
	}{
		{'あ', Hiragana, "hiragana"},
		{'ゝ', Hiragana, "hiragana iteration mark"},
		{'ア', Katakana, "katakana"},
		{'ー', Katakana, "prolonged sound mark"},
		{'・', Katakana, "katakana middle dot"},
		{'ｱ', Katakana, "halfwidth katakana"},
		{'東', Kanji, "kanji"},
		{'々', Kanji, "iteration mark 々"},
		{'〆', Kanji, "closing mark 〆"},
		{'〇', Kanji, "ideographic zero"},
		{'𠮟', Kanji, "extension B kanji"},
		{'﨑', Kanji, "compatibility ideograph"},
		{'A', Latin, "ascii upper"},
		{'z', Latin, "ascii lower"},
		{'Ａ', Latin, "fullwidth upper"},
		{'ｚ', Latin, "fullwidth lower"},
		{'é', Latin, "accented latin"},
		{'5', Digit, "ascii digit"},
		{'５', Digit, "fullwidth digit"},
		{'［', Symbol, "fullwidth bracket (between Ｚ and ａ in Unicode)"},
		{'＿', Symbol, "fullwidth underscore"},
		{'。', Symbol, "japanese period"},
		{'「', Symbol, "corner bracket"},
		{'!', Symbol, "ascii punctuation"},
		{' ', Space, "space"},
		{'\n', Space, "newline"},
		{'　', Space, "ideographic space"},
	}
	for _, tt := range tests {
		if got := Of(tt.r); got != tt.want {
			t.Errorf("Of(%q U+%04X) [%s] = %v, want %v", tt.r, tt.r, tt.desc, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	if Kanji.String() != "Kanji" || Space.String() != "Space" {
		t.Errorf("unexpected names: %s %s", Kanji, Space)
	}
	if CharType(99).String() != "Unknown" || CharType(-1).String() != "Unknown" {
		t.Error("out-of-range CharType should stringify as Unknown")
	}
}
