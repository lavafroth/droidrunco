package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/lavafroth/droidrunco/app"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	adb "github.com/zach-klippenstein/goadb"
)

const aapt string = "/data/local/tmp/aapt"

var device *adb.Device
var pkgs app.Apps
var db app.MetaDB
var client *adb.Adb

//go:embed aapt/*
var binaries embed.FS

//go:embed templates/index.html
var index []byte

//go:embed assets/*
var assets embed.FS

//go:embed knowledge.json
var rawKnowledge []byte

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

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		App, err := app.Unmarshal(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		filtered := pkgs.WithPackageOrLabel(strings.ToLower(App.Package))
		sort.Sort(filtered)
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

	if err := json.Unmarshal(rawKnowledge, &db); err != nil {
		log.Fatalln(err)
	}

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
	log.Info("Initializing package entries")
	refreshPackageList()
	go func() {
		for {
			refreshPackageList()
		}
	}()
	http.Handle("/public/", http.StripPrefix(strings.TrimRight("/public/", "/"), http.FileServer(http.FS(assets))))
	http.HandleFunc("/", handler)
	log.Info("Visit http://localhost:8080 to access the dashboard")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func worker(appChan, workChan chan *app.App) {
	for app := range workChan {
		out, err := device.RunCommand(fmt.Sprintf("%s d badging %s", aapt, app.Path))
		if err != nil {
			log.WithFields(log.Fields{
				"path": app.Path,
			}).Warnf("Failed to retrieve package label: %q", err)
		}

		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "application-label") {
				app.Label = line[19 : len(line)-1]
				app.HasLabel = true
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
	var newPkgs app.Apps
	appChan := make(chan *app.App)
	workChan := make(chan *app.App, 8)
	for i := 0; i < 8; i++ {
		go worker(appChan, workChan)
	}
	lines := strings.Split(out, "\n")
	newPkgCount := len(lines)
	go func() {
		for ; newPkgCount > 0; newPkgCount-- {
			app := <-appChan
			if k := db.Get(app.Package); k != nil {
				app.Description = k.Description
				app.Removal = k.Removal
			}

			if app.Description == "" {
				app.Description = "Description not yet available."
			}

			newPkgs = append(newPkgs, app)
		}
	}()

	for _, line := range lines {
		line := strings.Split(line, ":")[1]
		delim := strings.LastIndex(line, "=")
		path, pkg := line[:delim], line[delim+1:]
		if app := pkgs.Get(pkg); app != nil {
			newPkgs = append(newPkgs, app)
			newPkgCount--
			continue
		}
		workChan <- &app.App{Meta: app.Meta{Package: pkg}, Path: path, Enabled: true}
	}

	for _, app := range pkgs {
		if newPkgs.Get(app.Package) == nil {
			app.Enabled = false
			newPkgs = append(newPkgs, app)
		}
	}
	pkgs = newPkgs
}

func toggle(App *app.App) string {
	if App.Enabled {
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", App.Package))
		if err != nil {
			trace := fmt.Sprintf("Failed to run uninstall command on %s: %q", App.String(), err)
			log.Warn(trace)
			return trace
		}
		if !strings.Contains(out, "Success") {
			trace := fmt.Sprintf("Failed to uninstall %s", App.String())
			log.Warn(trace)
			return trace
		}

		App.Enabled = false

		trace := fmt.Sprintf("Successfully uninstalled %s", App.String())
		log.Info(trace)
		return trace
	}

	pathRe := regexp.MustCompile("path: (?P<path>.*.apk)")
	groupNames := pathRe.SubexpNames()
	out, err := device.RunCommand(fmt.Sprintf("pm dump %s", App.Package))
	if err != nil {
		trace := fmt.Sprintf("Failed to dump path for issuing reinstall command on %s: %q", App.String(), err)
		log.Warn(trace)
		return trace
	}
	path := ""
	for i, group := range pathRe.FindAllStringSubmatch(out, -1)[0] {
		if groupNames[i] == "path" {
			path = group
			break
		}
	}

	if path == "" {
		trace := fmt.Sprintf("Failed to find package path for %s: %q", App.String(), err)
		log.Warn(trace)
		return trace
	}

	out, err = device.RunCommand(fmt.Sprintf("pm install -r --user 0 %s", path))
	if err != nil {
		trace := fmt.Sprintf("Failed to run reinstall command on %s: %q", App.String(), err)
		log.Warn(trace)
		return trace
	}
	if !strings.Contains(out, "Success") {
		trace := fmt.Sprintf("Failed to reinstall %s", App.String())
		log.Warn(trace)
		return trace
	}

	App.Enabled = true

	trace := fmt.Sprintf("Successfully reinstalled %s", App.String())
	log.Info(trace)
	return trace
}
