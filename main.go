package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	adb "github.com/zach-klippenstein/goadb"
)

type Knowledge struct {
	Package     string `json:"id"`
	Description string `json:"description"`
	Removal     string `json:"removal"`
}

type App struct {
	Package     string `json:"pkg"`
	Description string `json:"description"`
	Removal     string `json:"removal"`
	Path     string `json:"-"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	HasLabel bool   `json:"-"`
}

type Apps []*App
type KnowledgeBase []*Knowledge

const aapt string = "/data/local/tmp/aapt"

var searchQuery string
var device *adb.Device
var pkgs Apps
var knowledgeBase KnowledgeBase
var client *adb.Adb

//go:embed aapt/*
var binaries embed.FS

//go:embed templates/index.html
var index []byte

//go:embed assets/*
var assets embed.FS

//go:embed knowledge.json
var rawKnowledge []byte

func (apps KnowledgeBase) WithPackageName(pkg string) *Knowledge {
	for _, app := range apps {
		if app.Package == pkg {
			return app

		}
	}
	return nil
}

func (apps Apps) WithPackageName(pkg string) *App {
	for _, app := range apps {
		if app.Package == pkg {
			return app

		}
	}
	return nil
}



func (app App) String() string {
	if app.HasLabel {
		return fmt.Sprintf("%s (%s)", app.Label, app.Package)
	}
	return app.Package
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
		var app App
		requestBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(requestBody, &app)
		query := strings.ToLower(app.Package)

		var apps Apps
		for _, v := range pkgs {
			if strings.Contains(strings.ToLower(v.Package), query) || strings.Contains(strings.ToLower(v.Label), query) {
				apps = append(apps, v)
			}
		}
		sort.Sort(apps)
		response, err := json.Marshal(apps)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	case http.MethodPatch:
		var app App
		requestBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(requestBody, &app)

		response, err := json.Marshal(map[string]string{"status": toggle(pkgs.WithPackageName(app.Package))})
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

	if err := json.Unmarshal(rawKnowledge, &knowledgeBase); err != nil {
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

func worker(appChan, workChan chan *App) {
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
	var newPkgs Apps
	appChan := make(chan *App)
	workChan := make(chan *App, 8)
	for i := 0; i < 8; i++ {
		go worker(appChan, workChan)
	}
	lines := strings.Split(out, "\n")
	newPkgCount := len(lines)
	go func() {
		for ; newPkgCount > 0; newPkgCount-- {
			app := <-appChan
				if k := knowledgeBase.WithPackageName(app.Package); k != nil {
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
		if app := pkgs.WithPackageName(pkg); app != nil {
			newPkgs = append(newPkgs, app)
			newPkgCount--
			continue
		}
		workChan <- &App{Package: pkg, Path: path, Enabled: true}
	}

	for _, app := range pkgs {
		if newPkgs.WithPackageName(app.Package) == nil {
			app.Enabled = false
			newPkgs = append(newPkgs, app)
		}
	}
	pkgs = newPkgs
}

func toggle(app *App) string {
	if app.Enabled {
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", app.Package))
		if err != nil {
			trace := fmt.Sprintf("Failed to run uninstall command on %s: %q", app.String(), err)
			log.Warn(trace)
			return trace
		}
		if !strings.Contains(out, "Success") {
			trace := fmt.Sprintf("Failed to uninstall %s", app.String())
			log.Warn(trace)
			return trace
		}

		app.Enabled = false

		trace := fmt.Sprintf("Successfully uninstalled %s", app.String())
		log.Info(trace)
		return trace
	}

	pathRe := regexp.MustCompile("path: (?P<path>.*.apk)")
	groupNames := pathRe.SubexpNames()
	out, err := device.RunCommand(fmt.Sprintf("pm dump %s", app.Package))
	if err != nil {
		trace := fmt.Sprintf("Failed to dump path for issuing reinstall command on %s: %q", app.String(), err)
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
		trace := fmt.Sprintf("Failed to find package path for %s: %q", app.String(), err)
		log.Warn(trace)
		return trace
	}

	out, err = device.RunCommand(fmt.Sprintf("pm install -r --user 0 %s", path))
	if err != nil {
		trace := fmt.Sprintf("Failed to run reinstall command on %s: %q", app.String(), err)
		log.Warn(trace)
		return trace
	}
	if !strings.Contains(out, "Success") {
		trace := fmt.Sprintf("Failed to reinstall %s", app.String())
		log.Warn(trace)
		return trace
	}

	app.Enabled = true

	trace := fmt.Sprintf("Successfully reinstalled %s", app.String())
	log.Info(trace)
	return trace
}
