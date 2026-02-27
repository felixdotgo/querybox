package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type App struct {
	App               *application.App
	MainWindow        *application.WebviewWindow
	ConnectionsWindow *application.WebviewWindow
	// PluginsWindow is a secondary window used to display the plugin list.
	PluginsWindow     *application.WebviewWindow
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
		URL:   "/#/connections",

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

		CloseButtonState: application.ButtonDisabled,
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

	// When the main window is closed we want the whole application to quit.
	// Closing the window alone is not sufficient on Windows/ Linux; the
	// process will continue running if there are other hidden windows or
	// background goroutines.  Attach an event listener so that a call to
	// CloseMainWindow or a user click of the close button triggers the
	// application shutdown.
	w.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// no need to cancel; we allow the window to close and then quit the app
		a.App.Quit()
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

// CloseMainWindow closes the main application window and initiates
// a full application shutdown.  Historically the UI called this method when
// the user selected Quit from the menu or pressed the window close button.
// Merely closing the webview did not terminate the Go process if there were
// other hidden windows or background services running, which led to the
// issue where the app would remain alive in the background.  We now call
// a.App.Quit() as well, which causes app.Run() to return and services to be
// torn down.
func (a *App) CloseMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.Close()
	}
	if a.App != nil {
		a.App.Quit()
	}
}

// ToggleFullScreenMainWindow toggles the main application window between fullscreen and windowed mode.
func (a *App) ToggleFullScreenMainWindow() {
	if a.MainWindow != nil {
		a.MainWindow.ToggleFullscreen()
	}
}

// Quit requests that the entire application shutdown.  In addition to closing
// the main window (which happens automatically), this causes app.Run() to
// return and triggers Shutdown on any bound services.
func (a *App) Quit() {
	if a.App != nil {
		a.App.Quit()
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

// NewPluginsWindow creates a new plugins window, mirroring the behaviour of the
// connections window.  The window is initially hidden and will be reused rather
// than re-created each time it is shown.
func (a *App) NewPluginsWindow() *application.WebviewWindow {
	w := a.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "plugins",
		Title: "Plugins",
		URL:   "/#/plugins",

		Frameless:     false,
		DisableResize: true,
		Hidden:        true,
		HideOnEscape:  true,
		MinWidth:      1024,

		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},

		CloseButtonState: application.ButtonDisabled,
	})

	// Intercept the close event and hide instead of destroying.
	w.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		a.ClosePluginsWindow()
	})

	// Prevent maximise/minimise just like the connections window.
	w.OnWindowEvent(events.Common.WindowMaximise, func(e *application.WindowEvent) { e.Cancel() })
	w.OnWindowEvent(events.Common.WindowMinimise, func(e *application.WindowEvent) { e.Cancel() })

	return w
}

// ShowPluginsWindow shows the plugins window, constructing it if necessary.
func (a *App) ShowPluginsWindow() {
	if a.PluginsWindow == nil {
		a.PluginsWindow = a.NewPluginsWindow()
	}
	a.PluginsWindow.Show()
	a.PluginsWindow.Focus()
}

// ClosePluginsWindow hides the plugins window.
func (a *App) ClosePluginsWindow() {
	if a.PluginsWindow != nil {
		a.PluginsWindow.SetAlwaysOnTop(false)
		// Hide rather than close; destroying the webview later causes crashes.
		a.PluginsWindow.Hide()
	}
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
