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
var listing, searchBox, logView *gocui.View
var device *adb.Device
var pkgs map[string]*App
var client *adb.Adb
var selection int

//go:embed aapt/*
var binaries embed.FS

type App struct {
	Path    string
	Package string
	Label   string
	Enabled bool
}

type Apps []*App

func (app *App) String() string {
	if len(app.Label) > 0 {
		return fmt.Sprintf("%s (%s)", app.Label, app.Package)
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

	pkgs = make(map[string]*App)
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
		log.Fatalf("failed to execute aapt: %q", err)
	}

	if strings.Contains(out, "not executable") {
		log.Fatalf("Failed to execute aapt: %q", out)
	}
	fmt.Print("Initializing package entries ... ")
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

func worker(appChan, workChan chan *App) {
	for app := range workChan {
		out, err := device.RunCommand(fmt.Sprintf("%s d badging %s", aapt, app.Path))
		if err != nil {
			fmt.Fprintf(logView, "failed to retrieve package label for apk at path %s: %q", app.Path, err)
		}
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "application-label") {
				app.Label = line[19 : len(line)-1]
				break
			}
		}

		appChan <- app
	}
}

func refreshPackageList() {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		log.Fatalf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	newPkgs := make(map[string]*App)
	appChan := make(chan *App)
	workChan := make(chan *App, 8)
	for i := 0; i < 8; i++ {
		go worker(appChan, workChan)
	}
	lines := strings.Split(out, "\n")
	newPkgCount := len(lines)
	go func() {
		for _, line := range lines {
			pkg := strings.Split(line, ":")[1]
			delim := strings.LastIndex(pkg, "=")
			path := pkg[:delim]
			packageName := pkg[delim+1:]
			if app, ok := pkgs[packageName]; ok {
				newPkgs[packageName] = app
				newPkgCount--
				continue
			}
			workChan <- &App{Package: packageName, Path: path, Enabled: true}
		}
	}()
	for ; newPkgCount > 0; newPkgCount-- {
		app := <-appChan
		newPkgs[app.Package] = app
	}
	for pkg, app := range pkgs {
		if _, ok := newPkgs[pkg]; !ok {
			app.Enabled = false
			newPkgs[pkg] = app
		}
	}
	pkgs = newPkgs
}

func toggleApp(app *App) {
	if app.Enabled {
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", app.Package))
		if err != nil {
			fmt.Fprintf(logView, "Failed to run uninstall command on %s: %q\n", app.String(), err)
			return
		}
		if !strings.Contains(out, "Success") {
			fmt.Fprintf(logView, "Failed to uninstall %s\n", app.String())
			return
		}
		fmt.Fprintf(logView, "Successfully uninstalled %s\n", app.String())
		app.Enabled = false
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
	app.Enabled = true
}

func search(query string) Apps {
	var result Apps
	query = strings.ToLower(strings.Trim(query, "\n\t "))
	for _, app := range pkgs {
		if strings.Contains(strings.ToLower(app.Label), query) || strings.Contains(strings.ToLower(app.Package), query) {
			result = append(result, app)
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
		case len(pkgs) > 0 && !app.Enabled:
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
