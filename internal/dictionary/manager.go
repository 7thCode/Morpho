package dictionary

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/7thCode/morpho/internal/hmm"
)

// Entry represents a single dictionary word entry.
type Entry struct {
	Surface   string `json:"surface"`
	Reading   string `json:"reading,omitempty"`
	POS       string `json:"pos"`
	POSDetail string `json:"pos_detail,omitempty"`
	Freq      int    `json:"freq"`
	// User marks entries registered by the user (as opposed to words
	// collected while training). Only user entries influence segmentation
	// and pin the POS during analysis.
	User bool `json:"user,omitempty"`
}

// ErrCorrupt is wrapped by the error Load returns when the file exists but is
// not a valid dictionary.
var ErrCorrupt = errors.New("dictionary file is corrupt")

// Dictionary holds a map of word entries and the associated HMM model.
type Dictionary struct {
	Entries map[string]*Entry `json:"entries"`
	Model   *hmm.Model        `json:"model"`
	// Counts are the trainer's accumulated counts, kept so that training
	// after a restart adds to the model instead of replacing it. Dictionaries
	// written by older versions have none.
	Counts  *hmm.Counts `json:"counts,omitempty"`
	Version int         `json:"version"`
}

// New creates and returns a new empty Dictionary.
func New() *Dictionary {
	return &Dictionary{
		Entries: make(map[string]*Entry),
		Model:   nil,
		Version: 1,
	}
}

// Load reads a Dictionary from a JSON file at path.
// If the file does not exist or is empty, it returns a new empty Dictionary (no error).
func Load(path string) (*Dictionary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return New(), nil
	}
	var d Dictionary
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCorrupt, path, err)
	}
	if d.Entries == nil {
		d.Entries = make(map[string]*Entry)
	}
	if d.Model != nil {
		d.Model.Prepare()
	}
	return &d, nil
}

// Save writes the Dictionary as JSON to the given path. The file is written
// to a temporary sibling and renamed into place, so a crash or full disk
// leaves the previous dictionary intact instead of a truncated one.
func (d *Dictionary) Save(path string) error {
	if path == "" {
		return errors.New("dictionary: no path to save to")
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// Update adds a new entry or increments the frequency of an existing entry.
func (d *Dictionary) Update(surface, pos string) {
	if entry, ok := d.Entries[surface]; ok {
		entry.Freq++
	} else {
		d.Entries[surface] = &Entry{
			Surface: surface,
			POS:     pos,
			Freq:    1,
		}
	}
}

// Lookup retrieves an entry by its surface form.
func (d *Dictionary) Lookup(surface string) (*Entry, bool) {
	entry, ok := d.Entries[surface]
	return entry, ok
}
