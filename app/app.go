package app

import (
	"fmt"

	"github.com/lavafroth/droidrunco/meta"
)

type App struct {
	Id string `json:"id"`
	meta.Meta
	Path     string `json:"-"`
	Enabled  bool   `json:"enabled"`
	HasLabel bool   `json:"-"`
}

type Apps map[string]*App

func (app *App) SetLabel(label string) {
	if label == "" {
		return
	}
	app.Label = label
	app.HasLabel = true
}

func (app *App) String() string {
	if app.HasLabel {
		return fmt.Sprintf("%s (%s)", app.Label, app.Id)
	}
	return app.Id
}
