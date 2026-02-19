package services

import "github.com/wailsapp/wails/v3/pkg/application"

type App struct {
	App               *application.App
	MainWindow        *application.WebviewWindow
	ConnectionsWindow *application.WebviewWindow
}

// NewAppService creates a new instance of the App service, which provides methods for controlling the main application window and the connections window.
func NewAppService() *App {
	return &App{}
}

// MaximiseMainWindow maximises the main application window to use the full screen size.
func (a *App) MaximiseMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.Maximise()
	}
}

// MinimiseMainWindow minimises the main application window.
func (a *App) MinimiseMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.Minimise()
	}
}

// CloseMainWindow closes the main application window.
func (a *App) CloseMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.Close()
	}
}

// ToggleFullScreenMainWindow toggles the main application window between fullscreen and windowed mode.
func (a *App) ToggleFullScreenMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.ToggleFullscreen()
	}
}

// ShowConnectionsWindow shows the connections window and brings it to the front.
func (a *App) ShowConnectionsWindow() {
	if a.ConnectionsWindow != nil {
		a.ConnectionsWindow.Show()
		a.ConnectionsWindow.Focus()
		a.ConnectionsWindow.SetAlwaysOnTop(true)
	}
}

// CloseConnectionsWindow hides the connections window and sends it to the back.
func (a *App) CloseConnectionsWindow() {
	if a.ConnectionsWindow != nil {
		a.ConnectionsWindow.SetAlwaysOnTop(false)
		// Hide the window instead of closing it. Closing destroys the underlying webview
		// which can cause "WEBKIT_IS_WEB_VIEW" assertion failures when reopened.
		a.ConnectionsWindow.Hide()
	}
}
