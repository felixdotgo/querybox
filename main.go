package main

import (
	"embed"
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/felixdotgo/querybox/services"
	"github.com/felixdotgo/querybox/services/pluginmgr"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	app := &services.App{}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app.App = application.New(application.Options{
		Name:        "querybox",
		Description: "A lightweight database management tool for executing and managing queries.",
		Services: []application.Service{
			application.NewService(services.NewConnectionService()),
			application.NewService(pluginmgr.New()),
			application.NewService(app), // Bind the App struct to allow frontend to call its methods (e.g. ShowConnections)
		},
		// Expose App methods (e.g. ShowConnections) to the frontend via bindings.
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.MainWindow = app.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "main",
		Title: "QueryBox",
		URL:   "/",
		DisableResize: false,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	// Create a connections window that is hidden by default. This window will be shown when the user clicks the "Connections" button in the UI.
	// or when there is no configured connection and the app needs to prompt the user to create one.
	app.ConnectionsWindow = app.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "connections",
		Title:  "Connections",
		URL:    "/connections",
		Hidden: true,
		DisableResize: true,
		MinWidth: 1024,
		MinHeight: 768,
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	// Run the application. This blocks until the application has been exited.
	err := app.App.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
