package hmm

import (
	"reflect"
	"testing"

	"github.com/7thCode/morpho/internal/tokenizer"
)

func sentenceSurfaces(sentences [][]tokenizer.Token) [][]string {
	var out [][]string
	for _, s := range sentences {
		var words []string
		for _, tok := range s {
			words = append(words, tok.Surface)
		}
		out = append(out, words)
	}
	return out
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want [][]string
	}{
		{"empty", "", nil},
		{"period", "私は行く。彼は来る。", [][]string{{"私", "は", "行", "く", "。"}, {"彼", "は", "来", "る", "。"}}},
		{"newline splits", "見出し\n本文が続く", [][]string{{"見出", "し"}, {"本文", "が", "続", "く"}}},
		{"plain space does not split", "a b", [][]string{{"a", "b"}}},
		{"ascii enders", "Hello! Really?", [][]string{{"Hello", "!"}, {"Really", "?"}}},
		{"closing quote stays attached", "「行く！」と言う", [][]string{{"「", "行", "く", "！」"}, {"と", "言", "う"}}},
		{"trailing text without ender", "終わり", [][]string{{"終", "わり"}}},
	}
	for _, tt := range tests {
		got := sentenceSurfaces(SplitSentences(tokenizer.Segment(tt.in)))
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestTrainOnTextHeuristic(t *testing.T) {
	tr := NewTrainer()
	n := TrainOnText("東京は首都です。\n大阪は都市です。", tr)
	if n != 2 {
		t.Fatalf("sentences = %d, want 2", n)
	}
	c := tr.Snapshot()
	if c.Initial[POSNoun] != 2 {
		t.Errorf("initial noun count = %v, want 2 (newline/period must start new sentences)", c.Initial[POSNoun])
	}
}

func TestTrainOnTextAnnotated(t *testing.T) {
	tr := NewTrainer()
	corpus := "猿/名詞 も/助詞 木/名詞 から/助詞 落ちる/動詞 。/記号\n" +
		"これは普通の行です。\n" + // plain line: heuristic
		"http://x/名詞 は/助詞\n" + // "/" inside word, POS is after the last "/"
		"a/notapos b/名詞\n" // one field invalid -> whole line treated as plain text
	n := TrainOnText(corpus, tr)
	c := tr.Snapshot()

	if c.Emission[POSVerb]["落ちる"] != 1 {
		t.Errorf("annotated word 落ちる should be trained as 動詞: %v", c.Emission[POSVerb])
	}
	if c.Emission[POSNoun]["http://x"] != 1 {
		t.Errorf("word containing '/' not parsed: %v", c.Emission[POSNoun])
	}
	if c.Emission[POSNoun]["a/notapos"] != 0 {
		t.Error("invalid POS should not be trained as annotation")
	}
	if n < 3 {
		t.Errorf("sentences = %d, want at least 3", n)
	}
}

func TestTrainOnTextUsesLexiconTags(t *testing.T) {
	lex := tokenizer.NewLexicon(map[string]string{"東京タワー": POSNoun, "ぬるぽ": "bogus"})
	tr := NewTrainer()
	TrainOnTextWithLexicon("東京タワーへ行く。", lex, tr)
	if tr.Snapshot().Emission[POSNoun]["東京タワー"] != 1 {
		t.Error("lexicon word should be trained whole with its tag")
	}

	tr = NewTrainer()
	TrainOnTextWithLexicon("ぬるぽ", lex, tr)
	// An invalid tag is ignored and the heuristic label is used instead.
	if _, ok := tr.Snapshot().Emission["bogus"]; ok {
		t.Error("invalid tag leaked into the model")
	}
}

func TestSymbolOnlySentencesAreNotTrained(t *testing.T) {
	tr := NewTrainer()
	if n := TrainOnText("。\n！！", tr); n != 0 {
		t.Errorf("symbol-only sentences trained: %d", n)
	}
}
