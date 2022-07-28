package main

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	adb "github.com/zach-klippenstein/goadb"
)

const aapt string = "/data/local/tmp/aapt"

var searchQuery string
var listing, searchBox *gocui.View
var device *adb.Device
var pkgs map[App]bool
var client *adb.Adb
var selection int

//go:embed aapt/*
var binaries embed.FS

type App struct {
	Package string
	Name    string
}

type Apps []App

func (app *App) String() string {
	if len(app.Name) > 0 {
		return fmt.Sprintf("%s (%s)", app.Name, app.Package)
	}
	return app.Package
}

func (apps Apps) Len() int {
	return len(apps)
}

func (apps Apps) Less(i, j int) bool {
	return strings.Compare(apps[i].String(), apps[j].String()) < 0
}

func (apps Apps) Swap(i, j int) {
	apps[i], apps[j] = apps[j], apps[i]
}

func push(local, remote string) error {
	localBytes, err := binaries.ReadFile("aapt/" + local)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %q", local, err)
	}
	remoteHandle, err := device.OpenWrite(remote, 0o755, time.Now())
	if err != nil {
		return fmt.Errorf("failed to open handle with write permissions on file %s: %q", remote, err)
	}
	defer remoteHandle.Close()
	if _, err := remoteHandle.Write(localBytes); err != nil {
		return fmt.Errorf("failed to copy data from local file handle to remote file: %q", err)
	}
	return nil
}

func main() {
	var err error
	pkgs = make(map[App]bool)
	client, err = adb.NewWithConfig(adb.ServerConfig{
		Port: 6000,
	})
	if err != nil {
		log.Fatalf("failed to start adb server: %q", err)
	}
	client.StartServer()
	defer client.KillServer()
	device = client.Device(adb.AnyDevice())

	binary := "x86"
	out, err := device.RunCommand("getprop ro.product.cpu.abi")
	if err != nil {
		log.Fatalf("failed to retrieve device architecture: %q, is the device connected?", err)
	}

	if strings.Contains(out, "arm") {
		binary = "arm"
	}

	if err := push(binary, aapt); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	out, err = device.RunCommand(aapt)
	if err != nil {
		log.Fatal("failed to execute aapt: %q", err)
	}

	if strings.Contains(out, "not executable") {
		log.Fatal("Failed to execute aapt: %q", out)
	}

	log.Println("Initializing package entries ...")
	updateCache()
	log.Println("Done.")

	go func() {
		for {
			// TODO: actually handle the error
			updateCache()
		}
	}()

	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatal("failed to create new console ui: %q", err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	if err != nil {
		log.Fatal(`failed to set "quit" keybind as ctrl + c: %q`, err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func updateCache() error {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		return fmt.Errorf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	refreshedPkgs := make(map[App]bool)
	for _, pkg := range strings.Split(out, "\n") {
		pkg = strings.Split(pkg, ":")[1]
		delim := strings.LastIndex(pkg, "=")
		app := App{Package: pkg[delim+1:]}
		out, err = device.RunCommand(fmt.Sprintf("%s d badging %s", aapt, pkg[:delim]))
		if err != nil {
			return fmt.Errorf("failed to refresh package list: %q", err)
		}
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "application-label") {
				app.Name = line[19 : len(line)-1]
				break
			}
		}
		refreshedPkgs[app] = true
	}
	pkgs = refreshedPkgs
	return nil
}

func uninstallApp(app App) {
	out, err := device.RunCommand(fmt.Sprintf("pm uninstall --user 0 %s", app.Package))
	if err != nil {
		log.Fatalf("failed to run uninstall command on %s: %q", app.String(), err)
	}
	if !strings.Contains(out, "Success") {
		log.Fatalf("failed to uninstall %s", app.String())
	}
	pkgs[app] = false
}

func search(query string) Apps {
	var result Apps
	query = strings.ToLower(strings.Trim(query, "\n\t "))
	for entry, ok := range pkgs {
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(entry.Name), query) || strings.Contains(strings.ToLower(entry.Package), query) {
			result = append(result, entry)
		}
	}
	sort.Sort(result)
	return result
}

func customEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
		selection = 0
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyCtrlL:
		v.Clear()
		_, cy := v.Cursor()
		v.SetCursor(1, cy)
	case key == gocui.KeyDelete:
		v.EditDelete(false)
	case key == gocui.KeyInsert:
		v.Overwrite = !v.Overwrite
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	case key == gocui.KeyArrowDown:
		selection++
	case key == gocui.KeyArrowUp:
		selection--
	}
	listing.Clear()
	apps := search(v.Buffer())
	if key == gocui.KeyEnter {
		uninstallApp(apps[selection])
		return
	}
	appCount := len(apps) - 1
	if selection > appCount {
		selection = appCount
	}
	if selection < 0 {
		selection = 0
	}
	listing.SetOrigin(0, selection)
	for i, app := range apps {
		if i == selection {
			fmt.Fprintf(listing, "\x1b[38;5;45m%s\x1b[0m\n", app.String())
			continue
		}
		fmt.Fprintln(listing, app.String())
	}
}

func layout(g *gocui.Gui) error {
	var err error
	X, Y := g.Size()
	if listing, err = g.SetView("listing", 0, 3, X-1, Y-1); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	listing.Wrap = true
	if searchBox, err = g.SetView("searchBox", 0, 0, X-1, 2); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	searchBox.Title = " Search app / package "
	searchBox.Editor = gocui.EditorFunc(customEditor)
	searchBox.Editable = true
	searchBox.Wrap = true
	searchBox.Autoscroll = true

	if _, err = g.SetCurrentView("searchBox"); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
