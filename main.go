package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	adb "github.com/zach-klippenstein/goadb"
)

var searchQuery string
var listing, searchBox *gocui.View
var device *adb.Device
var aaptPath string
var pkgs []string
var client *adb.Adb

func push(local, remote string) {
	localHandle, err := os.Open(local)
	if err != nil {
		panic(err)
	}
	defer localHandle.Close()
	remoteHandle, err := device.OpenWrite(remote, 0o755, time.Now())
	if err != nil {
		panic(err)
	}
	defer remoteHandle.Close()
	_, err = io.Copy(remoteHandle, localHandle)
	if err != nil {
		panic(err)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var err error
	client, err = adb.NewWithConfig(adb.ServerConfig{
		Port: 6000,
	})
	if err != nil {
		log.Fatal(err)
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
			time.Sleep(5 * time.Second)
		}
	}()

	g, err := gocui.NewGui(gocui.OutputNormal)
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
	pkgs = []string{}
	for _, pkg := range strings.Split(out, "\n") {
		pkg = strings.Split(pkg, ":")[1]
		delim := strings.LastIndex(pkg, "=")
		command := aaptPath + " d badging " + pkg[:delim]
		pkgName := pkg[delim+1:]
		out, err := device.RunCommand(command)
		if err != nil {
			return err
		}

		var friendlyName string
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "application-label") {
				friendlyName = line[19 : len(line)-1]
				break
			}
		}
		if len(friendlyName) > 0 {
			pkgs = append(pkgs, fmt.Sprintf("%s (%s)", friendlyName, pkgName))
		} else {
			pkgs = append(pkgs, pkgName)
		}
	}
	return nil
}

func search(query string) string {
	var result []string
	query = strings.ToLower(strings.Trim(query, "\n\t "))
	for _, entry := range pkgs {
		if strings.Contains(strings.ToLower(entry), query) {
			result = append(result, entry)
		}
	}
	return strings.Join(result, "\n")
}

func customEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
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
		v.EditNewLine()
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}
	listing.Clear()
	fmt.Fprintf(listing, search(v.Buffer()))
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
