package app

import (
	"fmt"

	"github.com/lavafroth/droidrunco/meta"
)

type App struct {
	Id string `json:"id"`
	meta.Meta
	Path    string `json:"-"`
	Enabled bool   `json:"enabled"`
}

type Apps map[string]*App

func (app *App) SetLabel(label string) {
	if label == "" || app.Label != "" {
		return
	}
	app.Label = label
}

func (app *App) String() string {
	if app.Label != "" {
		return fmt.Sprintf("%s (%s)", app.Label, app.Id)
	}
	return app.Id
}
