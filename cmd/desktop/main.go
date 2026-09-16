package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// buildMenu wires up the native "File" menu. The actual dialog/IO work
// happens in App (OpenTextFile / SaveSegmentedText); the menu just tells the
// frontend which action to perform, since the text currently being edited
// and the latest analysis result only exist in the frontend's state.
func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Open...", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "menu:open")
	})
	fileMenu.AddText("Write...", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "menu:write")
	})

	appMenu.Append(menu.EditMenu())
	appMenu.Append(menu.WindowMenu())

	return appMenu
}

func main() {
	app := NewApp()
	wails.Run(&options.App{
		Title:            "Morpho",
		Width:            960,
		Height:           680,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		Menu:             buildMenu(app),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		Mac:              &mac.Options{TitleBar: mac.TitleBarHiddenInset()},
	})
}
