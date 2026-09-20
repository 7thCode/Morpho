package tokenizer

import (
	"reflect"
	"testing"

	"github.com/7thCode/morpho/internal/chartype"
)

func surfaces(tokens []Token) []string {
	var out []string
	for _, t := range tokens {
		out = append(out, t.Surface)
	}
	return out
}

func TestSegment(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"東京都の天気", []string{"東京都", "の", "天気"}},
		{"ABC123", []string{"ABC", "123"}},
		{"時々、人々は", []string{"時々", "、", "人々", "は"}},
		{"らーめんを食べる", []string{"らーめんを", "食", "べる"}},
		{"東京タワー", []string{"東京", "タワー"}},
		{"１９８９年［注］", []string{"１９８９", "年", "［", "注", "］"}},
		{"a b", []string{"a", " ", "b"}},
	}
	for _, tt := range tests {
		if got := surfaces(Segment(tt.in)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Segment(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSegmentPositionsAreRuneOffsets(t *testing.T) {
	text := "東京は晴れ🙂ABC"
	runes := []rune(text)
	tokens := Segment(text)
	next := 0
	for _, tok := range tokens {
		if tok.StartPos != next {
			t.Errorf("token %q StartPos = %d, want contiguous %d", tok.Surface, tok.StartPos, next)
		}
		if got := string(runes[tok.StartPos:tok.EndPos]); got != tok.Surface {
			t.Errorf("runes[%d:%d] = %q, want %q", tok.StartPos, tok.EndPos, got, tok.Surface)
		}
		next = tok.EndPos
	}
	if next != len(runes) {
		t.Errorf("tokens cover %d runes, text has %d", next, len(runes))
	}
}

func TestSegmentWithLexicon(t *testing.T) {
	lex := NewLexicon(map[string]string{
		"東京タワー": "名詞",
		"行き":    "動詞",
		"":      "名詞", // ignored
		"a b":   "名詞", // ignored: contains whitespace
	})
	if lex.Len() != 2 {
		t.Fatalf("Len() = %d, want 2 (empty and whitespace-containing entries are dropped)", lex.Len())
	}

	tokens := SegmentWithLexicon("私は東京タワーへ行きたい", lex)
	if got, want := surfaces(tokens), []string{"私", "は", "東京タワー", "へ", "行き", "たい"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces = %q, want %q", got, want)
	}
	wantTags := map[string]string{"東京タワー": "名詞", "行き": "動詞"}
	for _, tok := range tokens {
		if wantTag := wantTags[tok.Surface]; tok.Tag != wantTag {
			t.Errorf("token %q Tag = %q, want %q", tok.Surface, tok.Tag, wantTag)
		}
	}

	// A lexicon word can start in the middle of what would otherwise be one
	// character-type run; the run is cut at the word boundary.
	if got, want := surfaces(SegmentWithLexicon("旅行き", lex)), []string{"旅", "行き"}; !reflect.DeepEqual(got, want) {
		t.Errorf("mid-run match = %q, want %q", got, want)
	}
}

func TestLexiconPrefersLongestMatch(t *testing.T) {
	lex := NewLexicon(map[string]string{"東京": "A", "東京都": "B"})
	tokens := SegmentWithLexicon("東京都", lex)
	if len(tokens) != 1 || tokens[0].Surface != "東京都" || tokens[0].Tag != "B" {
		t.Errorf("tokens = %+v, want single 東京都/B", tokens)
	}
}

func TestNilAndEmptyLexiconMatchSegment(t *testing.T) {
	text := "私は東京へ行く。"
	want := Segment(text)
	if got := SegmentWithLexicon(text, nil); !reflect.DeepEqual(got, want) {
		t.Errorf("nil lexicon changed the result: %+v", got)
	}
	if got := SegmentWithLexicon(text, NewLexicon(nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("empty lexicon changed the result: %+v", got)
	}
	var zero Lexicon
	if zero.Len() != 0 {
		t.Error("zero Lexicon should be empty")
	}
}

func TestSegmentTypes(t *testing.T) {
	tokens := Segment("猫はcat")
	types := []chartype.CharType{tokens[0].Type, tokens[1].Type, tokens[2].Type}
	want := []chartype.CharType{chartype.Kanji, chartype.Hiragana, chartype.Latin}
	if !reflect.DeepEqual(types, want) {
		t.Errorf("types = %v, want %v", types, want)
	}
}
