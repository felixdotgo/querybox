//go:build !darwin

package services

import "github.com/wailsapp/wails/v3/pkg/application"

func (a *App) NewAppMenu() *application.Menu {
	return nil
}
