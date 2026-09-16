//go:build js && wasm

// Command wasm compiles the morpho engine to WebAssembly and exposes it to
// JavaScript for the browser-based playground at docs/playground.html.
// It exports three global functions, each returning a JSON-encoded string:
//
//	morphoAnalyze(text)   -> {"ok":true,"morphemes":[...]}  | {"ok":false,"error":"..."}
//	morphoTrain(corpus)   -> {"ok":true}                    | {"ok":false,"error":"..."}
//	morphoStats()         -> {"word_count":N,"is_trained":bool,"pos_tags":[...]}
//
// The analyzer is created once at startup and lives only in the WASM
// instance's memory for the page's lifetime — nothing is persisted to disk
// (there is no real filesystem in the browser), so each page load starts
// from an untrained, empty dictionary exactly like a first run of the
// desktop app.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/7thCode/morpho"
)

var analyzer *morpho.Analyzer

func main() {
	analyzer = morpho.NewInMemory()

	js.Global().Set("morphoAnalyze", js.FuncOf(analyzeJS))
	js.Global().Set("morphoTrain", js.FuncOf(trainJS))
	js.Global().Set("morphoStats", js.FuncOf(statsJS))

	select {} // keep the Go runtime alive so JS can keep calling exported functions
}

func analyzeJS(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errResult("text argument required")
	}
	morphemes, err := analyzer.Analyze(args[0].String())
	if err != nil {
		return errResult(err.Error())
	}
	return okResult(map[string]any{"morphemes": morphemes})
}

func trainJS(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errResult("corpus argument required")
	}
	if err := analyzer.Train(args[0].String()); err != nil {
		return errResult(err.Error())
	}
	return okResult(nil)
}

func statsJS(_ js.Value, _ []js.Value) any {
	return mustJSON(map[string]any{
		"word_count": analyzer.WordCount(),
		"is_trained": analyzer.IsTrained(),
		"pos_tags":   analyzer.POSTags(),
	})
}

func okResult(extra map[string]any) string {
	result := map[string]any{"ok": true}
	for k, v := range extra {
		result[k] = v
	}
	return mustJSON(result)
}

func errResult(msg string) string {
	return mustJSON(map[string]any{"ok": false, "error": msg})
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"failed to encode result"}`
	}
	return string(b)
}
