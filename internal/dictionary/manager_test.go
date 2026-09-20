package dictionary

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/7thCode/morpho/internal/hmm"
)

func TestLoadMissingOrEmptyFileGivesEmptyDictionary(t *testing.T) {
	dir := t.TempDir()
	d, err := Load(filepath.Join(dir, "none.json"))
	if err != nil || len(d.Entries) != 0 {
		t.Fatalf("missing: %v %v", d, err)
	}
	empty := filepath.Join(dir, "empty.json")
	os.WriteFile(empty, nil, 0o644)
	if d, err := Load(empty); err != nil || d.Entries == nil {
		t.Fatalf("empty: %v %v", d, err)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	os.WriteFile(p, []byte(`{"entries": [`), 0o644)
	if _, err := Load(p); !errors.Is(err, ErrCorrupt) {
		t.Errorf("err = %v, want ErrCorrupt", err)
	}
}

func TestSaveLoadRoundTripKeepsCountsUserFlagAndPreparesModel(t *testing.T) {
	tr := hmm.NewTrainer()
	tr.AddSequence([]string{"猫", "が", "鳴く"}, []string{hmm.POSNoun, hmm.POSParticle, hmm.POSVerb})
	counts := tr.Snapshot()

	d := New()
	d.Model = tr.Build()
	d.Counts = &counts
	d.Entries["猫"] = &Entry{Surface: "猫", POS: hmm.POSNoun, Freq: 1, User: true}

	p := filepath.Join(t.TempDir(), "d.json")
	if err := d.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Counts == nil || got.Counts.Emission[hmm.POSNoun]["猫"] != 1 {
		t.Errorf("counts lost: %+v", got.Counts)
	}
	if !got.Entries["猫"].User {
		t.Error("user flag lost")
	}
	if got.Model.SmoothEmission(hmm.POSNoun, "犬") >= got.Model.SmoothEmission(hmm.POSNoun, "猫") {
		t.Error("loaded model does not behave like the saved one")
	}
}

func TestLoadsPreCountsFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.json")
	old := `{"entries":{"猫":{"surface":"猫","pos":"名詞","freq":3}},"model":{"initial":{"名詞":0},"transition":{},"emission":{"名詞":{"猫":0}},"pos_tags":["名詞"]},"version":1}`
	os.WriteFile(p, []byte(old), 0o644)
	d, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Counts != nil || d.Entries["猫"].User {
		t.Error("old files should load without counts and without user flags")
	}
	if v := d.Model.SmoothEmission(hmm.POSNoun, "犬"); v >= 0 {
		t.Errorf("legacy smoothing gave %v", v)
	}
}

func TestSaveFailureLeavesExistingFileIntact(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.json")
	d := New()
	d.Entries["a"] = &Entry{Surface: "a", POS: hmm.POSNoun, Freq: 1}
	if err := d.Save(p); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)

	// A model with NaN cannot be marshalled: Save must fail without
	// touching the good file or leaving temp files behind.
	d.Model = hmm.New()
	d.Model.Initial["x"] = math.NaN()
	if err := d.Save(p); err == nil {
		t.Fatal("expected marshal error")
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Error("failed Save modified the existing file")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
	if err := d.Save(""); err == nil {
		t.Error("Save(\"\") should fail")
	}
}

func TestUpdateAndLookup(t *testing.T) {
	d := New()
	d.Update("猫", hmm.POSNoun)
	d.Update("猫", hmm.POSVerb) // existing entry: only freq changes
	e, ok := d.Lookup("猫")
	if !ok || e.Freq != 2 || e.POS != hmm.POSNoun {
		t.Errorf("entry = %+v", e)
	}
}
