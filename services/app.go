package services

import "github.com/wailsapp/wails/v3/pkg/application"

type App struct {
	App               *application.App
	MainWindow        *application.WebviewWindow
	ConnectionsWindow *application.WebviewWindow
}

func (a *App) ShowConnectionsWindow() {
	if a.ConnectionsWindow != nil {
		a.ConnectionsWindow.Show()
		a.ConnectionsWindow.Focus()
		a.ConnectionsWindow.SetAlwaysOnTop(true)
	}
}

func (a *App) CloseConnectionsWindow() {
	if a.ConnectionsWindow != nil {
		a.ConnectionsWindow.SetAlwaysOnTop(false)
		// Hide the window instead of closing it. Closing destroys the underlying webview
		// which can cause "WEBKIT_IS_WEB_VIEW" assertion failures when reopened.
		a.ConnectionsWindow.Hide()
	}
}
