package main

import (
	"context"
	"sync"

	"github.com/7thCode/morpho"
)

type App struct {
	ctx      context.Context
	mu       sync.Mutex
	analyzer *morpho.Analyzer
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.analyzer, _ = morpho.New(resolveDictPath())
}

func (a *App) shutdown(_ context.Context) {}

func (a *App) Analyze(text string) ([]morpho.Morpheme, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.analyzer.Analyze(text)
}

func (a *App) Train(corpus string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.analyzer.Train(corpus); err != nil {
		return err
	}
	return a.analyzer.Save(resolveDictPath())
}

// Stats represents dictionary and HMM model status.
type Stats struct {
	WordCount int      `json:"word_count"`
	IsTrained bool     `json:"is_trained"`
	POSTags   []string `json:"pos_tags"`
}

// GetStats returns dictionary and HMM model status to the frontend.
func (a *App) GetStats() (Stats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.analyzer == nil {
		return Stats{}, nil
	}
	return Stats{
		WordCount: a.analyzer.WordCount(),
		IsTrained: a.analyzer.IsTrained(),
		POSTags:   a.analyzer.POSTags(),
	}, nil
}

// GetEntries returns all word entries stored in the dictionary to the frontend.
func (a *App) GetEntries() ([]morpho.DictEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.analyzer == nil {
		return nil, nil
	}
	return a.analyzer.Entries(), nil
}

// SaveWord adds or updates a word in the dictionary.
func (a *App) SaveWord(surface, pos string, freq int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.analyzer == nil {
		return nil
	}
	return a.analyzer.SaveWord(surface, pos, freq)
}

// DeleteWord removes a word from the dictionary.
func (a *App) DeleteWord(surface string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.analyzer == nil {
		return nil
	}
	return a.analyzer.DeleteWord(surface)
}


