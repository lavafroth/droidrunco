package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/meta"
	"net/http"
	"sort"
	"strings"
	"time"

	"log"
	adb "github.com/zach-klippenstein/goadb"
)

const extractor string = "/data/local/tmp/extractor"

var device *adb.Device
var pkgs app.Apps
var db meta.DB
var client *adb.Adb

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

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		App, err := app.Unmarshal(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		filtered := pkgs.WithPackageOrLabel(strings.ToLower(App.Package))
		response, err := json.Marshal(filtered)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	case http.MethodPatch:
		App, err := app.Unmarshal(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		response, err := json.Marshal(map[string]string{"status": toggle(pkgs.Get(App.Package))})
		if err != nil {
			log.Fatalln(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

func main() {
	var err error
	db, err = meta.Init()
	if err != nil {
		log.Fatal(err)
	}

	client, err = adb.NewWithConfig(adb.ServerConfig{
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
	time.Sleep(1 * time.Second)
	out, err = device.RunCommand(extractor)
	if err != nil {
		log.Fatalf("failed to execute extractor: %q", err)
	}

	if strings.Contains(out, "not executable") {
		log.Fatalf("Failed to execute extractor: %q", out)
	}
	log.Print("Initializing package entries")
	refreshPackageList()
	go func() {
		for {
			refreshPackageList()
		}
	}()
	http.Handle("/public/", http.StripPrefix(strings.TrimRight("/public/", "/"), http.FileServer(http.FS(assets))))
	http.HandleFunc("/", handler)
	log.Print("Visit http://localhost:8080 to access the dashboard")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func worker(work chan *app.App) {
	for app := range work{
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

	for _, line := range strings.Split(out, "\n") {
		line := strings.Split(line, ":")[1]
		delim := strings.LastIndex(line, "=")
		path, pkg := line[:delim], line[delim+1:]
		var App *app.App
		if App = pkgs.Get(pkg); App == nil {
			App = &app.App{Meta: meta.Meta{Package: pkg}, Path: path, Enabled: true}
			workChan <- App
		}
		fresh = append(fresh, App)
	}

	for _, app := range pkgs {
		if fresh.Get(app.Package) == nil {
			app.Enabled = false
			fresh = append(fresh, app)
		}
	}
	sort.Sort(fresh)
	pkgs = fresh
}

func toggle(App *app.App) string {
	// TODO: Send toast messages to the Web UI corresponding to each trace.
	if App.Enabled {
		// Isssue the uninstall command for the respective package
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", App.Package))
		if err != nil {
			trace := fmt.Sprintf("Failed to run uninstall command on %s: %q", App.String(), err)
			log.Print(trace)
			return trace
		}
		// If the output does not contain "Success",
		// we were unable to uninstall the app as user 0.
		if !strings.Contains(out, "Success") {
			trace := fmt.Sprintf("Failed to uninstall %s", App.String())
			log.Print(trace)
			// So we immediately return.
			return trace
		}

		// If we have successfully uninstalled
		// the app for user 0, we set the App's
		// Enabled field to false.
		App.Enabled = false

		trace := fmt.Sprintf("Successfully uninstalled %s", App.String())
		log.Print(trace)
		return trace
	}

	// If we are to re-enable a system package, we will dump its package info.
	out, err := device.RunCommand(fmt.Sprintf("pm dump %s", App.Package))
	if err != nil {
		trace := fmt.Sprintf("Failed to dump path for issuing reinstall command on %s: %q", App.String(), err)
		log.Print(trace)
		return trace
	}
	path := ""
	// We then look for a line that specifies
	// where the system app's installer resides.
	for _, line := range strings.Split(out, "\n") {
		line = strings.Trim(line, " \t")
		if strings.HasPrefix(line, "path: ") {
			// len("path: ") = 6
			// Index(".apk") + 4 to ensure we include the extension
			path = line[6: strings.Index(line, ".apk") + 4]
			break
		}
	}

	// If the path is empty,
	// it is probably not a system package
	// in which case, we can't proceed.
	if path == "" {
		trace := fmt.Sprintf("Failed to find package path for %s: %q", App.String(), err)
		log.Print(trace)
		// So, we return early.
		return trace
	}

	// If we have a valid path to the installer, we issue the reinstall command.
	out, err = device.RunCommand(fmt.Sprintf("pm install -r --user 0 %s", path))
	if err != nil {
		trace := fmt.Sprintf("Failed to run reinstall command on %s: %q", App.String(), err)
		log.Print(trace)
		return trace
	}

	// If the output does not contain "Success",
	// we were unable to reinstall the app as user 0.
	if !strings.Contains(out, "Success") {
		trace := fmt.Sprintf("Failed to reinstall %s", App.String())
		log.Print(trace)
		return trace
	}

	// If we have successfully reinstalled
	// the app for user 0, we set the App's
	// Enabled field to true.
	App.Enabled = true

	trace := fmt.Sprintf("Successfully reinstalled %s", App.String())
	log.Print(trace)
	return trace
}
