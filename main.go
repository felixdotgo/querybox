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
	// Register the structured log event so the binding generator can produce
	// a typed JS/TS API for it.
	application.RegisterEvent[services.LogEntry]("app:log")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	app := &services.App{}

	// Construct services before application.New so we can call SetApp afterwards.
	connSvc := services.NewConnectionService()
	mgr := pluginmgr.New()

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app.App = application.New(application.Options{
		Name:        "querybox",
		Description: "A lightweight database management tool for executing and managing queries.",
		Services: []application.Service{
			application.NewService(connSvc),
			application.NewService(mgr),
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

	// Inject the Wails app reference so services can emit log events to the frontend.
	connSvc.SetApp(app.App)
	mgr.SetApp(app.App)

	// Create default windows for the application.
	// The main window is the primary interface,
	// while the connections window is used for managing database connections.
	app.MainWindow = app.NewMainWindow()
	app.ConnectionsWindow = app.NewConnectionsWindow()

	// Run the application. This blocks until the application has been exited.
	err := app.App.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
