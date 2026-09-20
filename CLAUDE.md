# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test by name
go test -run TestAnalyzer ./...

# Build all packages
go build ./...

# Vet (lint)
go vet ./...

# Run the example
go run cmd/example/main.go
```

The library packages (root, `internal/...`) use only the standard library. `cmd/desktop` additionally depends on Wails (cgo; needs `frontend/dist`, kept in git only as `.gitkeep`).

## Architecture

This is a Japanese morphological analyzer implemented as a Go library (`github.com/7thCode/morpho`).

**Public API** (`morpho.go`): `Analyzer` with `New(dictPath)` / `OpenOrRecover` / `NewInMemory`, `Train(corpus)`, `Analyze(text)`, `Save(path)`, `SaveWord`/`DeleteWord`. The dictionary file (`dict.json`) persists word entries, the trained HMM model, and the trainer's counts (so training accumulates across restarts) as JSON. `Analyzer` is safe for concurrent use; `Save` is atomic (temp file + rename).

**Analysis pipeline** (text in → `[]Morpheme` out):

```
text
  → tokenizer.SegmentWithLexicon  // split at character-type boundaries; user-dictionary words kept whole (Token.Tag)
  → hmm.SplitSentences            // sentence enders (。！？!?) and line breaks
  → viterbi.Decode (per sentence) // optimal POS sequence via HMM; tagged tokens are pinned to their POS
  → []Morpheme{Surface, POS}
```

If no trained model is available, `Analyze` falls back to `hmm.LabelOf` (user-dictionary tag, else `hmm.InferPOS` heuristics by character type).

**Training pipeline** (corpus in → model stored in dictionary):

```
corpus
  → hmm.ParseCorpus          // per line: "word/POS ..." used as written, otherwise segment + label heuristically
  → trainer.AddSequence      // accumulate counts (restored from dictionary.Counts on load)
  → trainer.Build()          // smoothed log-probabilities
  → dictionary.Model/Counts  // stored on Analyzer, persisted via Save()
```

**Internal packages:**

| Package | Responsibility |
|---|---|
| `internal/chartype` | Maps Unicode runes to `CharType` (Hiragana, Katakana, Kanji, Latin, Digit, Symbol, Space) |
| `internal/tokenizer` | `Segment`/`SegmentWithLexicon` split text at `CharType` boundaries → `[]Token{Surface, Type, StartPos, EndPos (rune offsets), Tag}`; `Lexicon` holds user words |
| `internal/hmm` | `Model` stores initial/transition/emission log-probs plus the unknown-word model; `Trainer` accumulates counts (`Counts` snapshot for persistence) and builds the model; `ParseCorpus`/`SplitSentences` handle corpus and sentence parsing; `InferPOS` is the heuristic fallback |
| `internal/viterbi` | `Decode` runs Viterbi over `[]Token` using the HMM model, returning the best POS path |
| `internal/dictionary` | JSON-backed store of `Entry` records (surface, POS, freq, User) plus the embedded `*hmm.Model` and trainer `Counts`; atomic `Save` |

**HMM model details:** probabilities are stored in log-space (`math.Log`). `LogZero = -1e300` represents −∞. Initial/transition tables are Lidstone-smoothed (α=0.1) and dense; emissions use Witten-Bell discounting with a per-POS character-class distribution for unseen words (`UnknownMass`, `ClassEmission`). `数詞`/`記号` are always model states. Models saved by older versions (no `class_emission`) fall back to add-one smoothing in `SmoothEmission`, and Viterbi falls back to emission-only scoring when no valid transition exists. Call `Model.Prepare()` after building/loading a model (`Build` and `dictionary.Load` do).

**POS tags** are Japanese strings defined as constants in `internal/hmm`: `名詞`, `動詞`, `形容詞`, `助詞`, `助動詞`, `副詞`, `記号`, `数詞`, `外来語`, `未知語`.
