package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type App struct {
	App               *application.App
	MainWindow        *application.WebviewWindow
	ConnectionsWindow *application.WebviewWindow
}

// NewAppService creates a new instance of the App service, which provides methods for controlling the main application window and the connections window.
func NewAppService() *App {
	return &App{}
}

// NewConnectionsWindow creates a new connections window with specific options and event handlers to manage its behavior.
// The window is initially hidden and configured to prevent resizing, maximising, and minimising.
// It also includes OS-specific options for the title bar and backdrop.
func (a *App) NewConnectionsWindow() *application.WebviewWindow {
	w := a.App.Window.NewWithOptions(application.WebviewWindowOptions{
		// Required options
		Name:  "connections",
		Title: "Connections",
		URL:   "/connections",

		// Optional options
		Frameless:     false,
		DisableResize: true,
		Hidden:        true,
		HideOnEscape:  true,
		MinWidth:      1024,

		// OS-specific options
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	// Intercept the window close event to hide the window instead of closing it.
	w.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// Cancel the close event to prevent the window from being destroyed
		e.Cancel()
		// Instead of closing the window, we hide it and send it to the back.
		// This allows us to reuse the same window instance
		a.CloseConnectionsWindow()
	})

	// Intercept maximise and minimise events to prevent the connections window from being maximised or minimised.
	w.OnWindowEvent(events.Common.WindowMaximise, func(e *application.WindowEvent) {
		e.Cancel()
	})

	w.OnWindowEvent(events.Common.WindowMinimise, func(e *application.WindowEvent) {
		e.Cancel()
	})

	return w
}

// NewMainWindow creates a new main application window with specific options and returns it.
func (a *App) NewMainWindow() *application.WebviewWindow {
	w := a.App.Window.NewWithOptions(application.WebviewWindowOptions{
		// Required options
		Name:          "main",
		Title:         "QueryBox",
		URL:           "/",

		// Optional options
		MinWidth:      1280,
		MinHeight:     720,

		// OS-specific options
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	return w
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
	if a.ConnectionsWindow == nil {
		a.ConnectionsWindow = a.NewConnectionsWindow()
	}
	a.ConnectionsWindow.Show()
	a.ConnectionsWindow.Focus()
}

// OpenFileDialog opens a native file picker and returns the selected file path.
// Returns an empty string if the user cancels.
func (a *App) OpenFileDialog() (string, error) {
	return a.App.Dialog.OpenFile().
		SetTitle("Select SQLite Database File").
		CanChooseFiles(true).
		AddFilter("SQLite Database", "*.db;*.sqlite;*.sqlite3").
		AddFilter("All Files", "*").
		PromptForSingleSelection()
}

// CloseConnectionsWindow hides the connections window and sends it to the back.
func (a *App) CloseConnectionsWindow() {
	if a.ConnectionsWindow != nil {
		a.ConnectionsWindow.SetAlwaysOnTop(false)
		// Hide the window instead of closing it. Closing destroys the underlying webview
		// which can cause "WEBKIT_IS_WEB_VIEW" assertion failures when reopened.
		a.ConnectionsWindow.Hide()
		a.App.Event.Emit(EventConnectionsWindowClosed, true)
	}
}

// OpenURL opens the specified URL in the system's default browser.
func (a *App) OpenURL(url string) {
	a.App.Browser.OpenURL(url)
}

// ShowAboutDialog displays a native About dialog for the application.
func (a *App) ShowAboutDialog() {
	a.App.Dialog.Info().
		SetTitle("About QueryBox").
		SetMessage("QueryBox\nVersion 0.1.0\n\n© 2024 Felixdotgo").
		Show()
}
