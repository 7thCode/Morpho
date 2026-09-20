package hmm

// POS tag constants for Japanese.
const (
	POSNoun     = "名詞"
	POSVerb     = "動詞"
	POSAdj      = "形容詞"
	POSParticle = "助詞"
	POSAuxVerb  = "助動詞"
	POSAdverb   = "副詞"
	POSSymbol   = "記号"
	POSNumber   = "数詞"
	POSForeign  = "外来語"
	POSUnknown  = "未知語"
)

// AllPOS lists every POS tag the analyzer knows about.
var AllPOS = []string{
	POSNoun, POSVerb, POSAdj, POSParticle, POSAuxVerb,
	POSAdverb, POSSymbol, POSNumber, POSForeign, POSUnknown,
}

// IsValidPOS reports whether s is one of the known POS tags.
func IsValidPOS(s string) bool {
	for _, p := range AllPOS {
		if p == s {
			return true
		}
	}
	return false
}
