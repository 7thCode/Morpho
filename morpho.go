// Package morpho implements a Japanese morphological analyzer with zero
// external dependencies (Go standard library only).
//
// Analysis segments text at character-type boundaries (tokenizer.Segment)
// and tags each token's part of speech with a hidden Markov model decoded
// via Viterbi (viterbi.Decode). Before any training, Analyze falls back to
// heuristic POS inference based on character type and word-ending patterns.
//
//	analyzer, err := morpho.New("dict.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//	analyzer.Train("東京は日本の首都です。今日は良い天気ですね。")
//	morphemes, err := analyzer.Analyze("今日の東京は良い天気です。")
//	for _, m := range morphemes {
//		fmt.Printf("%s\t%s\n", m.Surface, m.POS)
//	}
//
// Words registered with SaveWord form a user dictionary: they are kept whole
// during segmentation (even across character types, e.g. 東京タワー) and
// pin the POS of matching tokens. Trained models and dictionary entries
// persist as JSON via Save, and reload automatically on the next New call
// against the same path. Training accumulates across restarts.
//
// An Analyzer is safe for concurrent use.
package morpho

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/7thCode/morpho/internal/dictionary"
	"github.com/7thCode/morpho/internal/hmm"
	"github.com/7thCode/morpho/internal/tokenizer"
	"github.com/7thCode/morpho/internal/viterbi"
)

// ErrCorruptDictionary is wrapped by the error New returns when the
// dictionary file exists but cannot be parsed. Use errors.Is to test for it,
// or OpenOrRecover to set the bad file aside and start fresh.
var ErrCorruptDictionary = dictionary.ErrCorrupt

// ErrInvalidInput is wrapped by errors caused by arguments the caller can fix
// (an unknown POS, a word containing whitespace, an empty corpus), as opposed
// to I/O failures. Use errors.Is to tell them apart.
var ErrInvalidInput = errors.New("invalid input")

// Morpheme represents a single morpheme with its surface form, reading, and POS information.
type Morpheme struct {
	Surface   string `json:"surface"`
	Reading   string `json:"reading,omitempty"`
	POS       string `json:"pos"`
	POSDetail string `json:"pos_detail,omitempty"`
}

// Analyzer performs Japanese morphological analysis.
type Analyzer struct {
	mu         sync.RWMutex
	dictPath   string
	dictionary *dictionary.Dictionary
	trainer    *hmm.Trainer
	lexicon    *tokenizer.Lexicon // user-registered words
}

// New creates an Analyzer, loading the dictionary from dictPath.
// If the file does not exist a fresh empty dictionary is used. A file that
// exists but is not a valid dictionary yields an error wrapping
// ErrCorruptDictionary.
func New(dictPath string) (*Analyzer, error) {
	dict, err := dictionary.Load(dictPath)
	if err != nil {
		return nil, err
	}
	return newAnalyzer(dictPath, dict), nil
}

// OpenOrRecover is like New, except that a corrupt dictionary file is renamed
// to "<dictPath>.corrupt-<timestamp>" and an empty dictionary is used, so a
// damaged file never locks the user out of the application. backup is the
// new name of the set-aside file, or "" if nothing was recovered.
func OpenOrRecover(dictPath string) (a *Analyzer, backup string, err error) {
	a, err = New(dictPath)
	if err == nil {
		return a, "", nil
	}
	if !errors.Is(err, ErrCorruptDictionary) {
		return nil, "", err
	}
	backup = fmt.Sprintf("%s.corrupt-%s", dictPath, time.Now().Format("20060102-150405"))
	if renameErr := os.Rename(dictPath, backup); renameErr != nil {
		return nil, "", fmt.Errorf("%w (and it could not be set aside: %v)", err, renameErr)
	}
	return newAnalyzer(dictPath, dictionary.New()), backup, nil
}

// NewInMemory creates an Analyzer with a fresh, empty dictionary that is
// never read from disk. Use this for ephemeral use — tests, or environments
// with no real filesystem such as WebAssembly in a browser — where New
// would otherwise fail trying to open a dictionary file. Analyze, Train,
// SaveWord, and DeleteWord work normally on the in-memory dictionary, and
// Save(path) writes it out on request; nothing is persisted automatically.
func NewInMemory() *Analyzer {
	return newAnalyzer("", dictionary.New())
}

func newAnalyzer(dictPath string, dict *dictionary.Dictionary) *Analyzer {
	trainer := hmm.NewTrainer()
	if dict.Counts != nil {
		trainer = hmm.NewTrainerFromCounts(*dict.Counts)
	}
	a := &Analyzer{dictPath: dictPath, dictionary: dict, trainer: trainer}
	a.rebuildLexicon()
	return a
}

// rebuildLexicon recomputes the user lexicon from the dictionary. Callers
// must hold a.mu for writing (or own the Analyzer exclusively).
func (a *Analyzer) rebuildLexicon() {
	words := make(map[string]string)
	for surface, e := range a.dictionary.Entries {
		if e.User {
			words[surface] = e.POS
		}
	}
	a.lexicon = tokenizer.NewLexicon(words)
}

// Train trains the HMM model from the given corpus text and updates the dictionary.
//
// Each corpus line is either plain text, whose POS labels are guessed from
// character type (user-dictionary words keep their registered POS), or a
// hand-labelled line of "word/POS" fields such as
//
//	東京/名詞 は/助詞 晴れ/名詞 。/記号
//
// which is learned exactly as written. Training accumulates: each call adds
// to the counts of earlier calls, including those from before a restart when
// the dictionary was saved. Dictionaries saved by earlier versions carry no
// counts, so the first Train on such a file starts the model afresh.
func (a *Analyzer) Train(corpus string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	sentences := hmm.ParseCorpus(corpus, a.lexicon)
	if len(sentences) == 0 {
		return fmt.Errorf("morpho: %w: corpus contains no trainable text", ErrInvalidInput)
	}
	for _, s := range sentences {
		a.trainer.AddSequence(s.Words, s.POS)
		for i, w := range s.Words {
			a.dictionary.Update(w, s.POS[i])
		}
	}

	a.dictionary.Model = a.trainer.Build()
	counts := a.trainer.Snapshot()
	a.dictionary.Counts = &counts
	return nil
}

// Analyze performs morphological analysis on the input text.
// If no trained model is available it falls back to heuristic POS inference.
func (a *Analyzer) Analyze(text string) ([]Morpheme, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sentences := hmm.SplitSentences(tokenizer.SegmentWithLexicon(text, a.lexicon))
	if len(sentences) == 0 {
		return nil, nil
	}

	model := a.dictionary.Model
	trained := model != nil && len(model.POSTags) > 0

	var morphemes []Morpheme
	for _, sentence := range sentences {
		var poses []string
		if trained {
			for _, r := range viterbi.Decode(sentence, model) {
				poses = append(poses, r.POS)
			}
		} else {
			for _, tok := range sentence {
				poses = append(poses, hmm.LabelOf(tok))
			}
		}
		for i, tok := range sentence {
			m := Morpheme{Surface: tok.Surface, POS: poses[i]}
			if tok.Tag != "" {
				if e := a.dictionary.Entries[tok.Surface]; e != nil {
					m.Reading, m.POSDetail = e.Reading, e.POSDetail
				}
			}
			morphemes = append(morphemes, m)
		}
	}
	return morphemes, nil
}

// Save persists the current dictionary (and model) to the given path.
func (a *Analyzer) Save(path string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.dictionary.Save(path)
}

// WordCount returns the number of entries in the dictionary.
func (a *Analyzer) WordCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.dictionary.Entries)
}

// IsTrained returns true if the analyzer has a trained HMM model.
func (a *Analyzer) IsTrained() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.dictionary.Model != nil && len(a.dictionary.Model.POSTags) > 0
}

// POSTags returns the list of POS tags used in the HMM model.
func (a *Analyzer) POSTags() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.dictionary.Model == nil {
		return nil
	}
	return append([]string(nil), a.dictionary.Model.POSTags...)
}

// DictEntry represents a single dictionary word entry.
type DictEntry struct {
	Surface   string `json:"surface"`
	Reading   string `json:"reading,omitempty"`
	POS       string `json:"pos"`
	POSDetail string `json:"pos_detail,omitempty"`
	Freq      int    `json:"freq"`
	// User is true for words registered with SaveWord.
	User bool `json:"user,omitempty"`
}

// Entries returns all dictionary entries, sorted by surface.
func (a *Analyzer) Entries() []DictEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	entries := make([]DictEntry, 0, len(a.dictionary.Entries))
	for _, entry := range a.dictionary.Entries {
		entries = append(entries, DictEntry{
			Surface:   entry.Surface,
			Reading:   entry.Reading,
			POS:       entry.POS,
			POSDetail: entry.POSDetail,
			Freq:      entry.Freq,
			User:      entry.User,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Surface < entries[j].Surface })
	return entries
}

// SaveWord registers a word in the user dictionary and persists the
// dictionary (unless the Analyzer is in-memory). From then on Analyze keeps
// the word whole and tags it pos. pos must be one of the known POS tags and
// surface must be non-empty and free of whitespace.
func (a *Analyzer) SaveWord(surface, pos string, freq int) error {
	if surface == "" || strings.TrimSpace(surface) != surface || strings.ContainsAny(surface, " \t\r\n\u3000") {
		return fmt.Errorf("morpho: %w: word %q must be non-empty and contain no whitespace", ErrInvalidInput, surface)
	}
	if !hmm.IsValidPOS(pos) {
		return fmt.Errorf("morpho: %w: unknown part of speech %q (valid: %s)", ErrInvalidInput, pos, strings.Join(hmm.AllPOS, ", "))
	}
	if freq < 0 {
		return fmt.Errorf("morpho: %w: negative frequency %d", ErrInvalidInput, freq)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	prev, had := a.dictionary.Entries[surface]
	a.dictionary.Entries[surface] = &dictionary.Entry{Surface: surface, POS: pos, Freq: freq, User: true}
	a.rebuildLexicon()

	if err := a.persist(); err != nil {
		if had {
			a.dictionary.Entries[surface] = prev
		} else {
			delete(a.dictionary.Entries, surface)
		}
		a.rebuildLexicon()
		return err
	}
	return nil
}

// DeleteWord removes a word entry from the dictionary and persists the
// dictionary (unless the Analyzer is in-memory).
func (a *Analyzer) DeleteWord(surface string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	prev, had := a.dictionary.Entries[surface]
	if !had {
		return nil
	}
	delete(a.dictionary.Entries, surface)
	a.rebuildLexicon()

	if err := a.persist(); err != nil {
		a.dictionary.Entries[surface] = prev
		a.rebuildLexicon()
		return err
	}
	return nil
}

// persist writes the dictionary to dictPath; a no-op for in-memory
// Analyzers. Callers must hold a.mu.
func (a *Analyzer) persist() error {
	if a.dictPath == "" {
		return nil
	}
	return a.dictionary.Save(a.dictPath)
}
