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
