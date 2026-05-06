package morpho_test

import (
	"os"
	"testing"

	"github.com/7thCode/morpho"
	"github.com/7thCode/morpho/internal/chartype"
	"github.com/7thCode/morpho/internal/hmm"
	"github.com/7thCode/morpho/internal/tokenizer"
	"github.com/7thCode/morpho/internal/viterbi"
)

// TestCharType verifies character type detection for various Unicode ranges.
func TestCharType(t *testing.T) {
	tests := []struct {
		r        rune
		expected chartype.CharType
		label    string
	}{
		{'あ', chartype.Hiragana, "hiragana"},
		{'ん', chartype.Hiragana, "hiragana n"},
		{'ア', chartype.Katakana, "katakana"},
		{'ン', chartype.Katakana, "katakana n"},
		{'東', chartype.Kanji, "kanji"},
		{'語', chartype.Kanji, "kanji 2"},
		{'A', chartype.Latin, "latin upper"},
		{'z', chartype.Latin, "latin lower"},
		{'0', chartype.Digit, "digit"},
		{'9', chartype.Digit, "digit 9"},
		{' ', chartype.Space, "space"},
		{'\t', chartype.Space, "tab"},
		{'。', chartype.Symbol, "japanese period"},
		{'、', chartype.Symbol, "japanese comma"},
	}

	for _, tt := range tests {
		got := chartype.Of(tt.r)
		if got != tt.expected {
			t.Errorf("chartype.Of(%q) [%s]: got %d, want %d", tt.r, tt.label, got, tt.expected)
		}
	}
}

// TestTokenize verifies that text is segmented at character-type boundaries.
func TestTokenize(t *testing.T) {
	// "東京都の天気" should produce: "東京都" (Kanji), "の" (Hiragana), "天気" (Kanji)
	tokens := tokenizer.Segment("東京都の天気")
	if len(tokens) < 3 {
		t.Errorf("expected at least 3 tokens, got %d: %+v", len(tokens), tokens)
	}

	// Verify surfaces
	expected := []string{"東京都", "の", "天気"}
	if len(tokens) == len(expected) {
		for i, tok := range tokens {
			if tok.Surface != expected[i] {
				t.Errorf("token[%d].Surface = %q, want %q", i, tok.Surface, expected[i])
			}
		}
	}

	// Test empty input
	empty := tokenizer.Segment("")
	if len(empty) != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", len(empty))
	}

	// Test mixed types
	mixed := tokenizer.Segment("ABC123")
	if len(mixed) != 2 {
		t.Errorf("expected 2 tokens for 'ABC123', got %d: %+v", len(mixed), mixed)
	}
}

// TestInferPOS verifies heuristic POS inference for common Japanese patterns.
func TestInferPOS(t *testing.T) {
	tests := []struct {
		surface  string
		ctype    chartype.CharType
		expected string
		label    string
	}{
		{"は", chartype.Hiragana, hmm.POSParticle, "particle wa"},
		{"が", chartype.Hiragana, hmm.POSParticle, "particle ga"},
		{"の", chartype.Hiragana, hmm.POSParticle, "particle no"},
		{"です", chartype.Hiragana, hmm.POSAuxVerb, "aux verb desu"},
		{"ます", chartype.Hiragana, hmm.POSAuxVerb, "aux verb masu"},
		{"東京", chartype.Kanji, hmm.POSNoun, "kanji noun"},
		{"コンピュータ", chartype.Katakana, hmm.POSForeign, "katakana foreign"},
		{"Hello", chartype.Latin, hmm.POSForeign, "latin foreign"},
		{"123", chartype.Digit, hmm.POSNumber, "digit number"},
		{"。", chartype.Symbol, hmm.POSSymbol, "symbol"},
	}

	for _, tt := range tests {
		tok := tokenizer.Token{Surface: tt.surface, Type: tt.ctype}
		got := hmm.InferPOS(tok)
		if got != tt.expected {
			t.Errorf("InferPOS(%q) [%s]: got %q, want %q", tt.surface, tt.label, got, tt.expected)
		}
	}
}

// TestHMMTrainer verifies that the trainer builds a valid non-nil model.
func TestHMMTrainer(t *testing.T) {
	corpus := "日本語の形態素解析は自然言語処理の基礎です。\n東京は日本の首都です。"
	trainer := hmm.NewTrainer()
	hmm.TrainOnText(corpus, trainer)
	model := trainer.Build()

	if model == nil {
		t.Fatal("Build() returned nil model")
	}
	if len(model.POSTags) == 0 {
		t.Error("model has no POS tags")
	}
	if len(model.Initial) == 0 {
		t.Error("model has no initial probabilities")
	}
	if len(model.Emission) == 0 {
		t.Error("model has no emission probabilities")
	}
}

// TestViterbi verifies that the Viterbi decoder returns results matching the token count.
func TestViterbi(t *testing.T) {
	corpus := "東京は日本の首都です。今日は良い天気です。"
	trainer := hmm.NewTrainer()
	hmm.TrainOnText(corpus, trainer)
	model := trainer.Build()

	tokens := tokenizer.Segment("東京は良い")
	// Filter spaces
	var nonSpace []tokenizer.Token
	for _, tok := range tokens {
		if tok.Type != chartype.Space {
			nonSpace = append(nonSpace, tok)
		}
	}

	if len(nonSpace) == 0 {
		t.Fatal("no non-space tokens to decode")
	}

	results := viterbi.Decode(nonSpace, model)
	if len(results) != len(nonSpace) {
		t.Errorf("Decode returned %d results, expected %d", len(results), len(nonSpace))
	}

	for i, r := range results {
		if r.Surface == "" {
			t.Errorf("result[%d] has empty surface", i)
		}
		if r.POS == "" {
			t.Errorf("result[%d] has empty POS", i)
		}
	}
}

// TestAnalyzer runs an end-to-end test: New, Train, Analyze.
func TestAnalyzer(t *testing.T) {
	// Use a temp file for the dictionary
	tmpFile, err := os.CreateTemp("", "morpho_test_dict_*.json")
	if err != nil {
		t.Fatal("failed to create temp file:", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	analyzer, err := morpho.New(tmpFile.Name())
	if err != nil {
		t.Fatal("New() failed:", err)
	}

	corpus := "日本語の形態素解析は自然言語処理の基礎です。\n東京は日本の首都です。\n今日は良い天気ですね。"
	if err := analyzer.Train(corpus); err != nil {
		t.Fatal("Train() failed:", err)
	}

	results, err := analyzer.Analyze("今日の東京は良い天気です。")
	if err != nil {
		t.Fatal("Analyze() failed:", err)
	}
	if len(results) == 0 {
		t.Fatal("Analyze() returned no results")
	}

	for i, m := range results {
		if m.Surface == "" {
			t.Errorf("morpheme[%d] has empty surface", i)
		}
		if m.POS == "" {
			t.Errorf("morpheme[%d] has empty POS", i)
		}
	}

	// Test Save
	if err := analyzer.Save(tmpFile.Name()); err != nil {
		t.Fatal("Save() failed:", err)
	}

	// Verify file was written
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal("stat after Save() failed:", err)
	}
	if info.Size() == 0 {
		t.Error("saved dictionary file is empty")
	}
}
