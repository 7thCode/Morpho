package dictionary

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/7thCode/morpho/internal/hmm"
)

// Entry represents a single dictionary word entry.
type Entry struct {
	Surface   string `json:"surface"`
	Reading   string `json:"reading,omitempty"`
	POS       string `json:"pos"`
	POSDetail string `json:"pos_detail,omitempty"`
	Freq      int    `json:"freq"`
}

// Dictionary holds a map of word entries and the associated HMM model.
type Dictionary struct {
	Entries map[string]*Entry `json:"entries"`
	Model   *hmm.Model        `json:"model"`
	Version int               `json:"version"`
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
		return nil, err
	}
	if d.Entries == nil {
		d.Entries = make(map[string]*Entry)
	}
	return &d, nil
}

// Save writes the Dictionary as JSON to the given path.
func (d *Dictionary) Save(path string) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
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
