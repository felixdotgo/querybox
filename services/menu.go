//go:build darwin

package services

import "github.com/wailsapp/wails/v3/pkg/application"

func (a *App) NewAppMenu() *application.Menu {
	menu := a.App.NewMenu()

	// Required on macOS: app name menu (About, Hide, Quit, etc.)
	menu.AddRole(application.AppMenu)

	// Edit menu — required for Cmd+C/V/X/A to work in text inputs on macOS.
	menu.AddRole(application.EditMenu)

	// File
	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("New Connection").OnClick(func(ctx *application.Context) {
		a.ShowConnectionsWindow()
	})
	// plugin listing window
	fileMenu.Add("Plugins").OnClick(func(ctx *application.Context) {
		a.ShowPluginsWindow()
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Quit QueryBox").SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
		a.CloseMainWindow()
	})

	// View
	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Toggle Fullscreen").
		SetAccelerator("Ctrl+Cmd+F").
		OnClick(func(ctx *application.Context) {
			a.ToggleFullScreenMainWindow()
		})
	viewMenu.Add("Toggle Logs").
		SetAccelerator("CmdOrCtrl+Shift+L").
		OnClick(func(ctx *application.Context) {
			a.App.Event.Emit(EventMenuLogsToggled, nil)
		})

	return menu
}
