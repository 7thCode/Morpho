package viterbi

import (
	"math"
	"testing"

	"github.com/7thCode/morpho/internal/hmm"
	"github.com/7thCode/morpho/internal/tokenizer"
)

func trainedModel() *hmm.Model {
	tr := hmm.NewTrainer()
	hmm.TrainOnText(`東京/名詞 は/助詞 日本/名詞 の/助詞 首都/名詞 です/助動詞 。/記号
大阪/名詞 も/助詞 都市/名詞 です/助動詞 。/記号
1989/数詞 年/名詞 に/助詞 起きた/動詞 。/記号
2001/数詞 年/名詞 に/助詞 起きた/動詞 。/記号`, tr)
	return tr.Build()
}

func poses(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.POS
	}
	return out
}

func TestDecodeEmptyInputs(t *testing.T) {
	if Decode(nil, trainedModel()) != nil {
		t.Error("nil tokens should give nil")
	}
	if Decode(tokenizer.Segment("猫"), nil) != nil {
		t.Error("nil model should give nil")
	}
	if Decode(tokenizer.Segment("猫"), hmm.New()) != nil {
		t.Error("model without states should give nil")
	}
}

func TestDecodeUnseenDigitsAreNumbers(t *testing.T) {
	// 2024 never appears in the corpus; the class-aware unknown-word model
	// must still label it 数詞 from its shape and context.
	got := poses(Decode(tokenizer.Segment("2024年に起きた"), trainedModel()))
	if got[0] != hmm.POSNumber {
		t.Errorf("unseen digits tagged %s, want 数詞 (all: %v)", got[0], got)
	}
}

func TestDecodeForcedTagOverridesModel(t *testing.T) {
	m := trainedModel()
	// Left alone, the seen noun 首都 is a 名詞. A tag pins it.
	tokens := tokenizer.Segment("首都")
	if got := Decode(tokens, m)[0].POS; got != hmm.POSNoun {
		t.Fatalf("baseline POS = %s, want 名詞", got)
	}
	tokens[0].Tag = hmm.POSVerb
	if got := Decode(tokens, m)[0].POS; got != hmm.POSVerb {
		t.Errorf("forced POS = %s, want 動詞", got)
	}
}

func TestDecodeForcedTagOutsideModelStatesStillLabels(t *testing.T) {
	m := trainedModel() // no 副詞 state
	if m.HasPOS(hmm.POSAdverb) {
		t.Fatal("test premise broken: model has 副詞")
	}
	tokens := tokenizer.Segment("東京")
	tokens[0].Tag = hmm.POSAdverb
	rs := Decode(tokens, m)
	if rs[0].POS != hmm.POSAdverb {
		t.Errorf("POS = %s, want the registered 副詞", rs[0].POS)
	}
}

func TestDecodeForcedTagInfluencesNeighbours(t *testing.T) {
	m := trainedModel()
	base := tokenizer.Segment("首都は")
	pinned := tokenizer.Segment("首都は")
	pinned[0].Tag = hmm.POSParticle
	a, b := poses(Decode(base, m)), poses(Decode(pinned, m))
	if b[0] != hmm.POSParticle || a[0] == b[0] {
		t.Fatalf("pin not applied: base=%v pinned=%v", a, b)
	}
}

func TestDecodeInvalidTagIsIgnored(t *testing.T) {
	m := trainedModel()
	tokens := tokenizer.Segment("首都")
	tokens[0].Tag = "bogus"
	if got := Decode(tokens, m)[0].POS; got != hmm.POSNoun {
		t.Errorf("POS = %s, want 名詞 (invalid tag must be ignored)", got)
	}
}

func TestDecodeLegacyModelWithSparseTables(t *testing.T) {
	// Hand-built model in the old format: only some transitions/initials.
	m := hmm.New()
	m.POSTags = []string{hmm.POSNoun, hmm.POSParticle}
	m.Initial[hmm.POSNoun] = math.Log(1)
	m.Transition[hmm.POSNoun] = map[string]float64{hmm.POSParticle: math.Log(1)}
	m.Emission[hmm.POSNoun] = map[string]float64{"猫": math.Log(1)}
	m.Emission[hmm.POSParticle] = map[string]float64{"が": math.Log(1)}

	tokens := tokenizer.Segment("猫が犬が")
	tokens[0].Tag = hmm.POSParticle // forces a state with no initial entry
	rs := Decode(tokens, m)
	if len(rs) != len(tokens) {
		t.Fatalf("got %d results for %d tokens", len(rs), len(tokens))
	}
	for _, r := range rs {
		if r.POS == "" {
			t.Errorf("empty POS in %+v", rs)
		}
	}
}
