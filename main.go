package main

import (
	"embed"
	"fmt"
	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/meta"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	adb "github.com/zach-klippenstein/goadb"
)

const extractor string = "/data/local/tmp/extractor"

var device *adb.Device
var pkgs app.Apps
var db meta.DB
var client *adb.Adb
var updated = false

//go:embed extractor/build/*
var binaries embed.FS

//go:embed templates/index.html
var index []byte

//go:embed assets/*
var assets embed.FS

func push(local, remote string) error {
	localBytes, err := binaries.ReadFile("extractor/build/" + local)
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

func WSLoopHandleFunc(path string, Fn func(conn *websocket.Conn) error) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed upgrading to websocket: %q", err)
		}
		defer conn.Close()
		for {
			if err := Fn(conn); err != nil {
				log.Print(err)
				return
			}
		}
	})
}

func main() {
	var err error
	db, err = meta.Init()
	if err != nil {
		log.Fatal(err)
	}

	client, err = adb.NewWithConfig(adb.ServerConfig{
		// Use the default ADB port.
		// This way, we don't have to adb kill-server
		// unless it was previously running on a different port.
		Port: 5037,
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

	if err := push(binary, extractor); err != nil {
		log.Fatal(err)
	}

	// It appears that immediately running the command to
	// execute the binary causes a file not found error.

	// Perhaps it is because pushing the file takes a while
	// and the file handle does not get closed immediately.

	// Thus, we add a 1 second delay.
	time.Sleep(1 * time.Second)

	out, err = device.RunCommand(extractor)
	if err != nil {
		log.Fatalf("failed to execute extractor: %q", err)
	}

	if strings.Contains(out, "not executable") {
		log.Fatalf("Failed to execute extractor: %q", out)
	}
	log.Print("Initializing package entries")

	// This first refresh is the most time consuming
	// as it has to index all the apps on the device
	refreshPackageList()

	// These subsequent calls are cheap, both in terms
	// of time as well as processing because we prune
	// all the packages previously seen.

	http.Handle("/public/", http.StripPrefix(strings.TrimRight("/public/", "/"), http.FileServer(http.FS(assets))))
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	})
	// TODO: WSLoopHandleFunc("/status", ...
	WSLoopHandleFunc("/list", func(conn *websocket.Conn) (err error) {
		refreshPackageList()
		if updated {
			if err = conn.WriteJSON(pkgs); err != nil {
				return fmt.Errorf("Failed writing fresh package list to websocket connection: %q", err)
			}
			updated = false
		}
		return
	})
	WSLoopHandleFunc("/patch", func(conn *websocket.Conn) (err error) {
		App := app.App{}
		if err = conn.ReadJSON(&App); err != nil {
			return fmt.Errorf("Failed to read patch query websocket connection: %q", err)
		}
		log.Print(toggle(pkgs.Get(App.Package)))
		// TODO: Send toast messages to the Web UI corresponding to each trace.
		updated = true
		return
	})
	log.Print("Visit http://localhost:8080 to access the dashboard")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func worker(work chan *app.App) {
	for app := range work {
		label, err := device.RunCommand(fmt.Sprintf("%s %s", extractor, app.Path))
		if err != nil {
			log.Printf("Failed to retrieve package label: %q, path: %s", err, app.Path)
		}

		app.SetLabel(strings.Trim(label, "\n"))

		if k := db.Get(app.Package); k != nil {
			app.Meta = *k
		}

		if app.Description == "" {
			app.Description = "Description not yet available."
		}
	}
}

func refreshPackageList() {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		log.Fatalf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	var fresh app.Apps
	workChan := make(chan *app.App, 8)
	for i := 0; i < 8; i++ {
		go worker(workChan)
	}

	gotFreshPackages := false

	for _, line := range strings.Split(out, "\n") {
		_, line, _ := strings.Cut(line, ":")
		i := strings.LastIndex(line, "=")
		path, pkg := line[:i], line[i+1:]

		// If we can already find the same package in the old list,
		// we don't bother looking up its label name.
		App := pkgs.Get(pkg)
		if App == nil {
			gotFreshPackages = true
			App = &app.App{Meta: meta.Meta{Package: pkg}, Path: path, Enabled: true}
			workChan <- App
		}
		fresh = append(fresh, App)
	}

	if !gotFreshPackages {
		return
	}
	
	updated = gotFreshPackages

	for _, app := range pkgs {
		// The app was previously enabled
		// but is no more in the new list.
		if fresh.Get(app.Package) == nil {
			// We can conclude that the
			// app has been disabled.
			app.Enabled = false
			fresh = append(fresh, app)
		}
	}
	sort.Sort(fresh)
	pkgs = fresh
}

func toggle(App *app.App) string {
	if App.Enabled {
		// Isssue the uninstall command for the respective package
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", App.Package))
		if err != nil {
			return fmt.Sprintf("Failed to run uninstall command on %s: %q", App.String(), err)
		}
		// If the output does not contain "Success",
		// we were unable to uninstall the app as user 0.
		if !strings.Contains(out, "Success") {
			// So we immediately return.
			return fmt.Sprintf("Failed to uninstall %s", App.String())
		}

		// If we have successfully uninstalled
		// the app for user 0, we set the App's
		// Enabled field to false.
		App.Enabled = false
		return fmt.Sprintf("Successfully uninstalled %s", App.String())
	}

	// If we are to re-enable a system package, we will dump its package info.
	out, err := device.RunCommand(fmt.Sprintf("pm dump %s", App.Package))
	if err != nil {
		return fmt.Sprintf("Failed to dump path for issuing reinstall command on %s: %q", App.String(), err)
	}
	path := ""
	// We then look for a line that specifies
	// where the system app's installer resides.
	for _, line := range strings.Split(out, "\n") {
		if _, after, found := strings.Cut(line, "path: "); found {
			// 4 since it is len(".apk")
			path = after[:strings.Index(after, ".apk")+4]
			break
		}
	}

	// If the path is empty,
	// it is probably not a system package
	// in which case, we can't proceed.
	if path == "" {
		// We return early.
		return fmt.Sprintf("Failed to find package path for %s: %q", App.String(), err)
	}

	// If we have a valid path to the installer, we issue the reinstall command.
	out, err = device.RunCommand(fmt.Sprintf("pm install -r --user 0 %s", path))
	if err != nil {
		return fmt.Sprintf("Failed to run reinstall command on %s: %q", App.String(), err)
	}

	// If the output does not contain "Success",
	// we were unable to reinstall the app as user 0.
	if !strings.Contains(out, "Success") {
		return fmt.Sprintf("Failed to reinstall %s", App.String())
	}

	// If we have successfully reinstalled
	// the app for user 0, we set the App's
	// Enabled field to true.
	App.Enabled = true
	return fmt.Sprintf("Successfully reinstalled %s", App.String())
}
