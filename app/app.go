package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/lavafroth/droidrunco/meta"
)

type App struct {
	meta.Meta
	Path     string `json:"-"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	HasLabel bool   `json:"-"`
}

type Apps []*App

func (app *App) SetLabel(label string) {
	if label == "" {
		return
	}
	app.Label = label
	app.HasLabel = true
}

func Unmarshal(r io.Reader) (*App, error) {
	var app App
	if err := json.NewDecoder(r).Decode(&app); err != nil {
		return nil, err
	}
	return &app, nil
}

func (app *App) String() string {
	if app.HasLabel {
		return fmt.Sprintf("%s (%s)", app.Label, app.Package)
	}
	return app.Package
}

func (apps Apps) Get(pkg string) *App {
	for _, app := range apps {
		if app.Package == pkg {
			return app
		}
	}
	return nil
}

func (apps Apps) Len() int {
	return len(apps)
}

func (apps Apps) Less(i, j int) bool {
	I, J := apps[i], apps[j]

	// Handle the exclusive cases where one app
	// has a label while the other does not.
	if I.HasLabel && !J.HasLabel {
		return true
	}
	if !I.HasLabel && J.HasLabel {
		return false
	}

	// By now, either both the apps will have labels
	// or none of them do.
	if I.HasLabel {
		return strings.Compare(I.Label, J.Label) < 0
	}
	return strings.Compare(I.Package, J.Package) < 0
}

func (apps Apps) Swap(i, j int) {
	apps[i], apps[j] = apps[j], apps[i]
}
