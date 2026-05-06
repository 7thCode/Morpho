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

No external dependencies — pure standard library.

## Architecture

This is a Japanese morphological analyzer implemented as a Go library (`github.com/7thCode/morpho`).

**Public API** (`morpho.go`): `Analyzer` with `New(dictPath)`, `Train(corpus)`, `Analyze(text)`, `Save(path)`. The dictionary file (`dict.json`) persists both word entries and the trained HMM model as JSON.

**Analysis pipeline** (text in → `[]Morpheme` out):

```
text
  → tokenizer.Segment        // split at character-type boundaries
  → viterbi.Decode           // find optimal POS sequence via HMM
  → []Morpheme{Surface, POS}
```

If no trained model is available, `Analyze` falls back to `hmm.InferPOS` (heuristic rules by character type).

**Training pipeline** (corpus in → model stored in dictionary):

```
corpus
  → tokenizer.Segment
  → hmm.TrainOnText          // sentence-split, infer POS heuristically, accumulate counts
  → trainer.Build()          // normalize counts to log-probabilities
  → dictionary.Model         // stored on Analyzer, persisted via Save()
```

**Internal packages:**

| Package | Responsibility |
|---|---|
| `internal/chartype` | Maps Unicode runes to `CharType` (Hiragana, Katakana, Kanji, Latin, Digit, Symbol, Space) |
| `internal/tokenizer` | `Segment` splits text at `CharType` boundaries → `[]Token{Surface, Type, StartPos, EndPos}` |
| `internal/hmm` | `Model` stores initial/transition/emission log-probs; `Trainer` accumulates counts and builds the model; `InferPOS` is the heuristic fallback |
| `internal/viterbi` | `Decode` runs Viterbi over `[]Token` using the HMM model, returning the best POS path |
| `internal/dictionary` | JSON-backed store of `Entry` records (surface, POS, freq) plus the embedded `*hmm.Model` |

**HMM model details:** probabilities are stored in log-space (`math.Log`). `LogZero = -1e300` represents −∞. Unknown-word smoothing uses Laplace (add-one) smoothing in `SmoothEmission`. The Viterbi implementation falls back to emission-only scoring when no valid transition exists.

**POS tags** are Japanese strings defined as constants in `internal/hmm`: `名詞`, `動詞`, `形容詞`, `助詞`, `助動詞`, `副詞`, `記号`, `数詞`, `外来語`, `未知語`.
