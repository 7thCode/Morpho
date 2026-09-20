package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/7thCode/morpho"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	// mu guards the fields below. The Analyzer is itself safe for concurrent
	// use, so methods only hold mu long enough to read them.
	mu            sync.Mutex
	analyzer      *morpho.Analyzer
	dictPath      string // "" when the dictionary could not be opened: nothing is persisted
	startupNotice string
}

// NewApp returns an App whose analyzer is an empty in-memory one, so bound
// methods called before (or without) startup never see a nil analyzer.
func NewApp() *App { return &App{analyzer: morpho.NewInMemory()} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg := loadConfig()

	analyzer, backup, err := morpho.OpenOrRecover(cfg.DictPath)

	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case err != nil:
		// Keep working, but do not write to a file we could not read: it may
		// still hold data that a save would destroy.
		a.analyzer = morpho.NewInMemory()
		a.dictPath = ""
		a.startupNotice = fmt.Sprintf("辞書ファイル(%s)を開けませんでした: %v。一時的な空の辞書で動作しており、変更は保存されません。", cfg.DictPath, err)
	case backup != "":
		a.analyzer, a.dictPath = analyzer, cfg.DictPath
		a.startupNotice = fmt.Sprintf("辞書ファイルが破損していたため %s に退避し、空の辞書で開始しました。", backup)
	default:
		a.analyzer, a.dictPath = analyzer, cfg.DictPath
	}
}

func (a *App) shutdown(_ context.Context) {}

// current returns the analyzer and dictionary path as of now.
func (a *App) current() (*morpho.Analyzer, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.analyzer, a.dictPath
}

// GetStartupNotice returns a message for the user if the dictionary could not
// be opened normally at startup, or "" if there is nothing to report.
func (a *App) GetStartupNotice() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startupNotice
}

func (a *App) Analyze(text string) ([]morpho.Morpheme, error) {
	analyzer, _ := a.current()
	return analyzer.Analyze(text)
}

func (a *App) Train(corpus string) error {
	analyzer, path := a.current()
	if err := analyzer.Train(corpus); err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	return analyzer.Save(path)
}

// Stats represents dictionary and HMM model status.
type Stats struct {
	WordCount int      `json:"word_count"`
	IsTrained bool     `json:"is_trained"`
	POSTags   []string `json:"pos_tags"`
}

// GetStats returns dictionary and HMM model status to the frontend.
func (a *App) GetStats() (Stats, error) {
	analyzer, _ := a.current()
	return Stats{
		WordCount: analyzer.WordCount(),
		IsTrained: analyzer.IsTrained(),
		POSTags:   analyzer.POSTags(),
	}, nil
}

// GetEntries returns all word entries stored in the dictionary to the frontend.
func (a *App) GetEntries() ([]morpho.DictEntry, error) {
	analyzer, _ := a.current()
	return analyzer.Entries(), nil
}

// SaveWord adds or updates a user-dictionary word. The analyzer persists it
// to the dictionary file it was opened from.
func (a *App) SaveWord(surface, pos string, freq int) error {
	analyzer, _ := a.current()
	return analyzer.SaveWord(surface, pos, freq)
}

// DeleteWord removes a word from the dictionary.
func (a *App) DeleteWord(surface string) error {
	analyzer, _ := a.current()
	return analyzer.DeleteWord(surface)
}

// GetDictPath returns the current dictionary path.
func (a *App) GetDictPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dictPath
}

// SetDictPath switches to the dictionary at path and remembers the choice.
// Unlike startup, a file the user explicitly picked is never moved aside: if
// it cannot be parsed the error is returned and the current dictionary stays.
func (a *App) SetDictPath(path string) error {
	analyzer, err := morpho.New(path)
	if err != nil {
		return err
	}
	if err := saveConfig(Config{DictPath: path}); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.analyzer = analyzer
	a.dictPath = path
	a.startupNotice = ""
	return nil
}

// SelectDictFile opens an OS file dialog for the user to select/create a dict.json file.
func (a *App) SelectDictFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "辞書ファイルを選択または新規作成 (dict.json)",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// OpenTextFile opens an OS file dialog and returns the contents of the
// selected text file. It returns an empty string if the user cancels.
func (a *App) OpenTextFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "テキストファイルを開く",
		Filters: []runtime.FileFilter{
			{DisplayName: "テキストファイル (*.txt)", Pattern: "*.txt"},
			{DisplayName: "すべてのファイル (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveSegmentedText writes already-segmented (分かち書き) text to a file
// chosen via an OS save dialog. It is a no-op if the user cancels.
func (a *App) SaveSegmentedText(text string) error {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "分かち書きを書き出す",
		DefaultFilename: "segmented.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "テキストファイル (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil || path == "" {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
