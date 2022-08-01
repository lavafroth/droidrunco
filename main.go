package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
    "io/ioutil"
    "bytes"

	"github.com/jroimartin/gocui"
	"github.com/shogo82148/androidbinary"
	"github.com/shogo82148/androidbinary/apk"
	adb "github.com/zach-klippenstein/goadb"
)

var searchQuery string
var listing, searchBox, logView *gocui.View
var device *adb.Device
var pkgs map[App]bool
var client *adb.Adb
var selection int
var resConfigEN *androidbinary.ResTableConfig

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

func main() {
	var err error
    resConfigEN = &androidbinary.ResTableConfig{
    Language: [2]uint8{uint8('e'), uint8('n')},
}

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
	fmt.Print("Refreshing package entries ... ")
	refreshPackageList()
	fmt.Println("done.")
	go func() {
		for {
			refreshPackageList()
		}
	}()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalf("failed to create new console ui: %q", err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	if err != nil {
		log.Fatalf(`failed to set "quit" keybind as ctrl + c: %q`, err)
	}
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func fetchLabel(path string) (label string) {
    fileStat, err := device.Stat(path)
    if err != nil {
        return
    }
    rawReader, err := device.OpenRead(path)
    if err != nil {
        return
    }
    defer rawReader.Close()
    rawBytes, err := ioutil.ReadAll(rawReader)
    if err != nil {
        return
    }
    apk, err := apk.OpenZipReader(bytes.NewReader(rawBytes), int64(fileStat.Size))
    if err != nil {
        return
    }
    defer apk.Close()
    label, err = apk.Label(resConfigEN)
    if err != nil {
        return
    }
    return
}

func refreshPackageList() {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		log.Fatalf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	newPkgs := make(map[App]bool)
    for _, pkg := range strings.Split(out, "\n") {
		pkg = strings.Split(pkg, ":")[1]
		delim := strings.LastIndex(pkg, "=")
		newPkgs[App{Name:fetchLabel(pkg[:delim]), Package: pkg[delim+1:]}] = true
	}
	for pkg, _ := range pkgs {
		if _, ok := newPkgs[pkg]; !ok {
			newPkgs[pkg] = false
		}
	}
	pkgs = newPkgs
}

func toggleApp(app App) {
	if state, _ := pkgs[app]; state {
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall --user 0 -k %s", app.Package))
		if err != nil {
			fmt.Fprintf(logView, "Failed to run uninstall command on %s: %q\n", app.String(), err)
			return
		}
		if !strings.Contains(out, "Success") {
			fmt.Fprintf(logView, "Failed to uninstall %s\n", app.String())
			return
		}
		fmt.Fprintf(logView, "Successfully uninstalled %s\n", app.String())
		pkgs[app] = false
		return
	}
	out, err := device.RunCommand(fmt.Sprintf("pm install-existing %s", app.Package))
	if err != nil {
		fmt.Fprintf(logView, "Failed to run reinstall command on %s: %q\n", app.String(), err)
		return
	}
	if !strings.Contains(out, "Success") {
		fmt.Fprintf(logView, "Failed to reinstall %s\n", app.String())
		return
	}
	fmt.Fprintf(logView, "Successfully reinstalled %s\n", app.String())
	pkgs[app] = true
}

func search(query string) Apps {
	var result Apps
	query = strings.ToLower(strings.Trim(query, "\n\t "))
	for entry, _ := range pkgs {
		if strings.Contains(strings.ToLower(entry.Name), query) || strings.Contains(strings.ToLower(entry.Package), query) {
			result = append(result, entry)
		}
	}
	sort.Sort(result)
	return result
}

func refreshListing() {
	listing.Clear()
	apps := search(searchBox.Buffer())
	appCount := len(apps) - 1
	if selection > appCount {
		selection = appCount
	}
	if selection < 0 {
		selection = 0
	}
	listing.SetOrigin(0, selection)
	for i, app := range apps {
		switch {
		case i == selection:
			fmt.Fprintf(listing, "\x1b[36;1m%s\x1b[0m\n", app.String())
		case len(pkgs) > 0 && !pkgs[app]:
			fmt.Fprintf(listing, "\x1b[31;4m%s\x1b[0m\n", app.String())
		default:
			fmt.Fprintln(listing, app.String())
		}
	}
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
	case key == gocui.KeyEnter:
		apps := search(searchBox.Buffer())
		toggleApp(apps[selection])
	}
	refreshListing()
}

func layout(g *gocui.Gui) error {
	var err error
	X, Y := g.Size()
	if listing, err = g.SetView("listing", 0, 3, X-1, int(4*Y/5)-1); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	listing.Wrap = true

	if logView, err = g.SetView("logView", 0, int(4*Y/5), X-1, Y-1); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	logView.Wrap = true
	logView.Autoscroll = true
	logView.Title = " Logs "
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
