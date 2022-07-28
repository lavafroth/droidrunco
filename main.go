package main

import (
	"fmt"
	"io"
	"log"
	"sort"
	"os"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	adb "github.com/zach-klippenstein/goadb"
)

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

var searchQuery string
var listing, searchBox *gocui.View
var device *adb.Device
var aaptPath string
var pkgs map[App]bool
var client *adb.Adb
var selection int

func push(local, remote string) {
	localHandle, err := os.Open(local)
	checkErr(err)
	defer localHandle.Close()
	remoteHandle, err := device.OpenWrite(remote, 0o755, time.Now())
	checkErr(err)
	defer remoteHandle.Close()
	_, err = io.Copy(remoteHandle, localHandle)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	pkgs = make(map[App]bool)
	var err error
	client, err = adb.NewWithConfig(adb.ServerConfig{
		Port: 6000,
	})
	if err != nil {
		log.Fatalf("failed to start adb server: %q",err)
	}
	client.StartServer()
	defer client.KillServer()
	device = client.Device(adb.AnyDevice())

	binary := "aapt-x86-pie"
	out, err := device.RunCommand("getprop ro.product.cpu.abi")
	checkErr(err)

	if strings.Contains(out, "arm") {
		binary = "aapt-arm-pie"
	}

	remotePath := "/data/local/tmp/"
	aaptPath = remotePath + binary
	push(binary, aaptPath)

	time.Sleep(1 * time.Second)

	out, err = device.RunCommand(aaptPath)
	checkErr(err)

	if strings.Contains(out, "not executable") {
		log.Fatal("Failed to execute aapt")
	}

	go func() {
		for {
			updateCache()
		}
	}()

	g, err := gocui.NewGui(gocui.Output256)
	checkErr(err)
	defer g.Close()

	g.SetManagerFunc(layout)
	checkErr(g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit))

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func updateCache() error {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		return err
	}
	out = strings.Trim(out, "\n\t ")
	refreshedPkgs := make(map[App]bool)
	for _, pkg := range strings.Split(out, "\n") {
		pkg = strings.Split(pkg, ":")[1]
		delim := strings.LastIndex(pkg, "=")
		app := App{Package: pkg[delim+1:]}
		if len(pkgs) != 0 {
			command := aaptPath + " d badging " + pkg[:delim]
			out, err := device.RunCommand(command)
			if err != nil {
				return err
			}

			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "application-label") {
					app.Name = line[19 : len(line)-1]
					break
				}
			}
		}
		refreshedPkgs[app] = true

	}
	pkgs = refreshedPkgs
	return nil
}

func uninstallApp(app App) {
	command := fmt.Sprintf("pm uninstall --user 0 %s", app.Package)
	out, err := device.RunCommand(command)
	checkErr(err)
	if strings.Contains(out, "Success") {
		// Remove the app from the set
		pkgs[app] = false
	}
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
flushed := false
result := []App{}
flush := func() {
	listing.Clear()
	result = search(v.Buffer())
}

	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
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
	case key == gocui.KeyEnter:
		flush()
		flushed = true
		uninstallApp(result[selection])
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	case key == gocui.KeyArrowDown:
		selection++
	case key == gocui.KeyArrowUp:
		if selection > 0 {
			selection--
		}
	}
	if !flushed {
		flush()
	}
	if selection >= len(result) {
		selection = len(result) - 1
	}
	for idx, entry := range result {
		if idx == selection {
			fmt.Fprintf(listing, "\x1b[38;5;45m%s\x1b[0m\n", entry.String())
		} else {
			fmt.Fprintln(listing, entry.String())
		}
	}
}

func layout(g *gocui.Gui) error {
	var err error
	maxX, maxY := g.Size()
	if listing, err = g.SetView("listing", 0, 3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	if searchBox, err = g.SetView("searchBox", 0, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	searchBox.Title = " Search app / package "
	searchBox.Editor = gocui.EditorFunc(customEditor)
	searchBox.Editable = true
	searchBox.Wrap = true
	searchBox.Autoscroll = true

	if _, err := g.SetCurrentView("searchBox"); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
