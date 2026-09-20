package morpho_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/7thCode/morpho"
)

func surfacesOf(ms []morpho.Morpheme) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Surface
	}
	return out
}

func posOf(t *testing.T, ms []morpho.Morpheme, surface string) string {
	t.Helper()
	for _, m := range ms {
		if m.Surface == surface {
			return m.POS
		}
	}
	t.Fatalf("%q not found in %v", surface, surfacesOf(ms))
	return ""
}

const testCorpus = "日本語の形態素解析は自然言語処理の基礎です。\n東京は日本の首都です。\n今日は良い天気ですね。\n" +
	"1989年に事件が起きた。\n2001年にも事件が起きた。\n"

func TestTrainAccumulatesAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dict.json")

	a, err := morpho.New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Train("東京は日本の首都です。"); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(path); err != nil {
		t.Fatal(err)
	}

	b, err := morpho.New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Train("大阪は西日本の都市です。"); err != nil {
		t.Fatal(err)
	}

	// The model must still know 東京 from the first session: had the second
	// Train replaced the model, 東京 would be an unseen word, and this
	// exact-match score would fall back to the unknown-word estimate.
	got, err := b.Analyze("東京は首都です。")
	if err != nil || len(got) == 0 {
		t.Fatalf("Analyze: %v %v", got, err)
	}
	// Retraining the same corpus in one session must produce the same model
	// as training it across two sessions.
	c := morpho.NewInMemory()
	_ = c.Train("東京は日本の首都です。")
	_ = c.Train("大阪は西日本の都市です。")
	pathB, pathC := filepath.Join(t.TempDir(), "b.json"), filepath.Join(t.TempDir(), "c.json")
	if err := b.Save(pathB); err != nil {
		t.Fatal(err)
	}
	if err := c.Save(pathC); err != nil {
		t.Fatal(err)
	}
	bb, _ := os.ReadFile(pathB)
	cc, _ := os.ReadFile(pathC)
	if string(bb) != string(cc) {
		t.Error("two-session training and single-session training produced different dictionaries")
	}
}

func TestSaveIsDeterministic(t *testing.T) {
	var files [][]byte
	for i := 0; i < 5; i++ {
		a := morpho.NewInMemory()
		if err := a.Train(testCorpus); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(t.TempDir(), "d.json")
		if err := a.Save(p); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(p)
		files = append(files, b)
	}
	for i := 1; i < len(files); i++ {
		if string(files[i]) != string(files[0]) {
			t.Fatal("dictionary JSON differs between identical trainings")
		}
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	a, _ := morpho.New(filepath.Join(dir, "dict.json"))
	if err := a.Train(testCorpus); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(filepath.Join(dir, "dict.json")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("directory has %d files, want only dict.json: %v", len(entries), entries)
	}
}

func TestUserDictionaryAffectsAnalysis(t *testing.T) {
	a := morpho.NewInMemory()
	text := "私は東京タワーへ行く"

	before, _ := a.Analyze(text)
	if !reflect.DeepEqual(surfacesOf(before)[2:4], []string{"東京", "タワー"}) {
		t.Fatalf("premise: without the word, 東京タワー splits: %v", surfacesOf(before))
	}

	if err := a.SaveWord("東京タワー", "名詞", 1); err != nil {
		t.Fatal(err)
	}
	after, _ := a.Analyze(text)
	if posOf(t, after, "東京タワー") != "名詞" {
		t.Errorf("registered word not kept whole/tagged: %v", after)
	}

	// Registered POS wins even after training, whatever the model thinks.
	if err := a.Train(testCorpus); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveWord("東京タワー", "動詞", 1); err != nil {
		t.Fatal(err)
	}
	trained, _ := a.Analyze(text)
	if posOf(t, trained, "東京タワー") != "動詞" {
		t.Errorf("registered POS not applied after training: %v", trained)
	}

	if err := a.DeleteWord("東京タワー"); err != nil {
		t.Fatal(err)
	}
	gone, _ := a.Analyze(text)
	for _, m := range gone {
		if m.Surface == "東京タワー" {
			t.Error("deleted word still kept whole")
		}
	}
}

func TestUserEntriesAreFlagged(t *testing.T) {
	a := morpho.NewInMemory()
	_ = a.Train("東京は首都です。")
	_ = a.SaveWord("ぬるぽ", "名詞", 1)
	var user, trained int
	for _, e := range a.Entries() {
		if e.User {
			user++
			if e.Surface != "ぬるぽ" {
				t.Errorf("unexpected user entry %q", e.Surface)
			}
		} else {
			trained++
		}
	}
	if user != 1 || trained == 0 {
		t.Errorf("user=%d trained=%d", user, trained)
	}
}

func TestSaveWordValidation(t *testing.T) {
	a := morpho.NewInMemory()
	for _, tt := range []struct {
		surface, pos string
		freq         int
	}{
		{"", "名詞", 1},
		{"a b", "名詞", 1},
		{" x", "名詞", 1},
		{"東京", "noun", 1},
		{"東京", "", 1},
		{"東京", "名詞", -1},
	} {
		if err := a.SaveWord(tt.surface, tt.pos, tt.freq); err == nil {
			t.Errorf("SaveWord(%q,%q,%d) succeeded, want error", tt.surface, tt.pos, tt.freq)
		}
	}
	if a.WordCount() != 0 {
		t.Errorf("invalid words were stored: %d entries", a.WordCount())
	}
}

func TestSaveWordRollsBackWhenPersistFails(t *testing.T) {
	// The directory does not exist, so writing the dictionary fails.
	a, err := morpho.New(filepath.Join(t.TempDir(), "missing", "dict.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.SaveWord("東京タワー", "名詞", 1); err == nil {
		t.Fatal("expected persist error")
	}
	if a.WordCount() != 0 {
		t.Error("failed SaveWord left the entry in memory")
	}
	ms, _ := a.Analyze("東京タワー")
	if len(ms) != 2 {
		t.Errorf("failed SaveWord still affects analysis: %v", surfacesOf(ms))
	}
}

func TestUserDictionarySurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dict.json")
	a, _ := morpho.New(path)
	if err := a.SaveWord("東京タワー", "名詞", 1); err != nil {
		t.Fatal(err)
	}
	b, err := morpho.New(path)
	if err != nil {
		t.Fatal(err)
	}
	ms, _ := b.Analyze("東京タワー")
	if len(ms) != 1 {
		t.Errorf("user word lost on reload: %v", surfacesOf(ms))
	}
}

func TestTrainAnnotatedCorpus(t *testing.T) {
	a := morpho.NewInMemory()
	// Heuristics would call 落ちる a noun-less hiragana/kanji mix; the
	// annotation says 動詞, and 猿 is a 名詞 only because we say so.
	if err := a.Train("猿/名詞 も/助詞 木/名詞 から/助詞 落ちる/動詞 。/記号\n猿/名詞 も/助詞 木/名詞 から/助詞 落ちる/動詞 。/記号"); err != nil {
		t.Fatal(err)
	}
	ms, _ := a.Analyze("猿も木から落ちる。")
	want := map[string]string{"猿": "名詞", "も": "助詞", "木": "名詞", "から": "助詞", "。": "記号"}
	for s, p := range want {
		if got := posOf(t, ms, s); got != p {
			t.Errorf("%s: %s, want %s", s, got, p)
		}
	}
}

func TestTrainEmptyCorpusIsAnError(t *testing.T) {
	a := morpho.NewInMemory()
	for _, c := range []string{"", "  \n\n", "。。。"} {
		if err := a.Train(c); err == nil {
			t.Errorf("Train(%q) succeeded, want error", c)
		}
	}
	if a.IsTrained() {
		t.Error("empty training produced a model")
	}
}

func TestAnalyzeLinesAreIndependentSentences(t *testing.T) {
	a := morpho.NewInMemory()
	if err := a.Train(testCorpus); err != nil {
		t.Fatal(err)
	}
	joined, _ := a.Analyze("東京は首都です\n大阪は都市です")
	first, _ := a.Analyze("東京は首都です")
	second, _ := a.Analyze("大阪は都市です")
	want := append(append([]morpho.Morpheme{}, first...), second...)
	if !reflect.DeepEqual(joined, want) {
		t.Errorf("newline is not a sentence boundary:\n got %v\nwant %v", joined, want)
	}
}

func TestHeldOutTagging(t *testing.T) {
	a := morpho.NewInMemory()
	if err := a.Train(testCorpus); err != nil {
		t.Fatal(err)
	}
	// None of these numbers or the symbols appear verbatim in the corpus.
	ms, err := a.Analyze("2024年に事件が起きた。")
	if err != nil {
		t.Fatal(err)
	}
	if got := posOf(t, ms, "2024"); got != "数詞" {
		t.Errorf("unseen number tagged %s, want 数詞", got)
	}
	if got := posOf(t, ms, "。"); got != "記号" {
		t.Errorf("。 tagged %s, want 記号", got)
	}
	ms, _ = a.Analyze("「2024」です。")
	if got := posOf(t, ms, "「"); got != "記号" {
		t.Errorf("unseen bracket tagged %s, want 記号", got)
	}
}

func TestUntrainedFallbackHonoursUserWords(t *testing.T) {
	a := morpho.NewInMemory()
	_ = a.SaveWord("ぬるぽ", "名詞", 1)
	ms, _ := a.Analyze("ぬるぽ")
	if len(ms) != 1 || ms[0].POS != "名詞" {
		t.Errorf("got %v", ms)
	}
}

func TestPOSTagsReturnsCopy(t *testing.T) {
	a := morpho.NewInMemory()
	_ = a.Train(testCorpus)
	tags := a.POSTags()
	if len(tags) == 0 {
		t.Fatal("no tags")
	}
	tags[0] = "changed"
	if a.POSTags()[0] == "changed" {
		t.Error("POSTags exposes internal slice")
	}
}

func TestCorruptDictionary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dict.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := morpho.New(path); !errors.Is(err, morpho.ErrCorruptDictionary) {
		t.Fatalf("New err = %v, want ErrCorruptDictionary", err)
	}

	a, backup, err := morpho.OpenOrRecover(path)
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" || !strings.Contains(backup, ".corrupt-") {
		t.Errorf("backup = %q", backup)
	}
	if data, _ := os.ReadFile(backup); string(data) != "{not json" {
		t.Errorf("backup does not hold the original bytes: %q", data)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("corrupt file still at original path")
	}
	if a.WordCount() != 0 {
		t.Error("recovered analyzer is not empty")
	}
	// The recovered analyzer is usable and persists over the original path.
	if err := a.SaveWord("猫", "名詞", 1); err != nil {
		t.Fatal(err)
	}
	if _, backup, err := morpho.OpenOrRecover(path); err != nil || backup != "" {
		t.Errorf("healthy file: backup=%q err=%v", backup, err)
	}
}

func TestOpenOrRecoverPropagatesOtherErrors(t *testing.T) {
	dir := t.TempDir() // a directory where a file is expected: read fails, not corrupt
	if _, _, err := morpho.OpenOrRecover(dir); err == nil {
		t.Error("expected an error opening a directory as a dictionary")
	}
}

func TestShippedDictionaryStillLoads(t *testing.T) {
	// The dict.json shipped in the repo must load and analyze.
	a, err := morpho.New("dict.json")
	if err != nil {
		t.Fatal(err)
	}
	if !a.IsTrained() {
		t.Skip("dict.json has no model")
	}
	ms, err := a.Analyze("今日の東京は良い天気です。")
	if err != nil || len(ms) == 0 {
		t.Fatalf("Analyze: %v %v", ms, err)
	}
}

func TestConcurrentUse(t *testing.T) {
	a := morpho.NewInMemory()
	if err := a.Train(testCorpus); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				switch (i + j) % 5 {
				case 0:
					_ = a.Train("東京は日本の首都です。")
				case 1:
					_ = a.SaveWord("東京タワー", "名詞", j)
				case 2:
					_ = a.DeleteWord("東京タワー")
				case 3:
					_, _ = a.Analyze("私は東京タワーへ行く。2024年です。")
				default:
					_ = a.Entries()
					_ = a.POSTags()
					_ = a.IsTrained()
					_ = a.WordCount()
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestNumbersAndSymbolsTaggedWithoutBeingInCorpus(t *testing.T) {
	// A corpus with no digits and no punctuation at all.
	a := morpho.NewInMemory()
	if err := a.Train("吾輩は猫である\n名前はまだ無い"); err != nil {
		t.Fatal(err)
	}
	ms, _ := a.Analyze("吾輩は2024年に猫を見た。")
	if got := posOf(t, ms, "2024"); got != "数詞" {
		t.Errorf("2024 tagged %s, want 数詞", got)
	}
	if got := posOf(t, ms, "。"); got != "記号" {
		t.Errorf("。 tagged %s, want 記号", got)
	}
}
