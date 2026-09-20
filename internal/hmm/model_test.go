package hmm

import (
	"math"
	"reflect"
	"sort"
	"testing"
)

func trainedModel() *Model {
	tr := NewTrainer()
	tr.AddSequence([]string{"東京", "は", "日本", "の", "首都", "です", "。"},
		[]string{POSNoun, POSParticle, POSNoun, POSParticle, POSNoun, POSAuxVerb, POSSymbol})
	tr.AddSequence([]string{"1989", "年", "に", "起きた", "。"},
		[]string{POSNumber, POSNoun, POSParticle, POSVerb, POSSymbol})
	tr.AddSequence([]string{"テスト", "を", "する", "。"},
		[]string{POSForeign, POSParticle, POSVerb, POSSymbol})
	return tr.Build()
}

func TestBuildPOSTagsSorted(t *testing.T) {
	for i := 0; i < 20; i++ {
		m := trainedModel()
		if !sort.StringsAreSorted(m.POSTags) {
			t.Fatalf("POSTags not sorted: %v", m.POSTags)
		}
	}
}

func TestBuildTablesAreDenseAndNormalized(t *testing.T) {
	m := trainedModel()
	var initSum float64
	for _, from := range m.POSTags {
		initSum += math.Exp(m.LogInitial(from))
		var rowSum float64
		for _, to := range m.POSTags {
			lp := m.LogTransition(from, to)
			if lp <= LogZero {
				t.Fatalf("transition %s->%s is impossible; smoothing should make it improbable", from, to)
			}
			rowSum += math.Exp(lp)
		}
		if math.Abs(rowSum-1) > 1e-9 {
			t.Errorf("transition row %s sums to %v, want 1", from, rowSum)
		}
	}
	if math.Abs(initSum-1) > 1e-9 {
		t.Errorf("initial distribution sums to %v, want 1", initSum)
	}
}

func TestSmoothEmissionUsesCharacterClass(t *testing.T) {
	m := trainedModel()

	// An unseen number is likelier to be 数詞 than 助詞...
	if m.SmoothEmission(POSNumber, "2024") <= m.SmoothEmission(POSParticle, "2024") {
		t.Error("unseen digits should favour 数詞 over 助詞")
	}
	// ...and an unseen kanji word likelier to be 名詞 than 数詞.
	if m.SmoothEmission(POSNoun, "天気") <= m.SmoothEmission(POSNumber, "天気") {
		t.Error("unseen kanji should favour 名詞 over 数詞")
	}
	// Seen words keep their trained probability.
	if got, want := m.SmoothEmission(POSNoun, "東京"), m.LogEmission(POSNoun, "東京"); got != want {
		t.Errorf("seen word: SmoothEmission = %v, want LogEmission %v", got, want)
	}
	// A seen word is likelier than an unseen word of the same class.
	if m.SmoothEmission(POSNoun, "東京") <= m.SmoothEmission(POSNoun, "大阪") {
		t.Error("seen word should outscore unseen word")
	}
	// Unknown POS still yields a finite score.
	if v := m.SmoothEmission("nonexistent", "x"); v <= LogZero || math.IsNaN(v) {
		t.Errorf("unknown POS gave %v", v)
	}
}

func TestLegacyModelFallsBackToAddOne(t *testing.T) {
	m := New()
	m.POSTags = []string{POSNoun}
	m.Emission[POSNoun] = map[string]float64{"猫": math.Log(1)}
	// No ClassEmission: original add-one estimate 1/(|inner|+vocab+1).
	want := math.Log(1.0 / (1 + 1 + 1))
	if got := m.SmoothEmission(POSNoun, "犬"); math.Abs(got-want) > 1e-12 {
		t.Errorf("legacy smoothing = %v, want %v", got, want)
	}
}

func TestPrepareIsIdempotentAndUnpreparedModelIsReadOnly(t *testing.T) {
	m := trainedModel()
	m.Prepare()
	v := m.vocabCount
	if v == 0 {
		t.Fatal("Prepare did not compute vocabulary size")
	}
	m.Prepare()
	if m.vocabCount != v {
		t.Errorf("vocabCount changed: %d -> %d", v, m.vocabCount)
	}

	// A model that skipped Prepare (e.g. hand-built) must not mutate itself
	// when read, or concurrent Analyze calls would race.
	raw := New()
	raw.Emission["a"] = map[string]float64{"x": 0}
	_ = raw.SmoothEmission("a", "y")
	if raw.vocabCount != 0 {
		t.Error("SmoothEmission cached into an unprepared model")
	}
}

func TestCountsSnapshotRoundTrip(t *testing.T) {
	tr := NewTrainer()
	tr.AddSequence([]string{"猫", "が", "鳴く"}, []string{POSNoun, POSParticle, POSVerb})

	snap := tr.Snapshot()
	restored := NewTrainerFromCounts(snap)
	if !reflect.DeepEqual(restored.Snapshot(), snap) {
		t.Fatal("restored trainer's counts differ from snapshot")
	}

	// Snapshot is a deep copy: mutating it must not affect the trainer.
	snap.Emission[POSNoun]["猫"] = 99
	if tr.Snapshot().Emission[POSNoun]["猫"] != 1 {
		t.Error("Snapshot shares storage with the trainer")
	}

	// Continuing training on the restored trainer accumulates.
	restored.AddSequence([]string{"猫"}, []string{POSNoun})
	if restored.Snapshot().Emission[POSNoun]["猫"] != 2 {
		t.Error("restored trainer did not accumulate")
	}
	if !reflect.DeepEqual(trainedModelFrom(tr), trainedModelFrom(NewTrainerFromCounts(tr.Snapshot()))) {
		t.Error("model built from restored counts differs")
	}
}

func trainedModelFrom(tr *Trainer) *Model { return tr.Build() }

func TestEmptyTrainerBuildsEmptyModel(t *testing.T) {
	m := NewTrainer().Build()
	if len(m.POSTags) != 0 {
		t.Errorf("POSTags = %v, want empty", m.POSTags)
	}
}

func TestShapeOnlyTagsExistEvenWhenUnseen(t *testing.T) {
	tr := NewTrainer()
	tr.AddSequence([]string{"猫", "が", "鳴く"}, []string{POSNoun, POSParticle, POSVerb})
	m := tr.Build()

	if !m.HasPOS(POSNumber) || !m.HasPOS(POSSymbol) {
		t.Fatalf("POSTags = %v, want 数詞 and 記号 present", m.POSTags)
	}
	if m.SmoothEmission(POSNumber, "2024") <= m.SmoothEmission(POSNoun, "2024") {
		t.Error("unseen digits should favour the (untrained) 数詞 over 名詞")
	}
	if m.SmoothEmission(POSSymbol, "。") <= m.SmoothEmission(POSParticle, "。") {
		t.Error("unseen symbol should favour the (untrained) 記号 over 助詞")
	}
	if m.SmoothEmission(POSNumber, "猫") >= m.SmoothEmission(POSNoun, "猫") {
		t.Error("seen kanji word must still prefer its trained tag over 数詞")
	}
	for _, from := range m.POSTags {
		for _, to := range m.POSTags {
			if m.LogTransition(from, to) <= LogZero {
				t.Errorf("transition %s->%s missing", from, to)
			}
		}
	}
}
