package main

import (
	"html/template"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"net/http"

	"github.com/gin-gonic/gin"
	adb "github.com/zach-klippenstein/goadb"
)

const aapt string = "/data/local/tmp/aapt"

var searchQuery string
var device *adb.Device
var pkgs map[string]*App
var client *adb.Adb

//go:embed aapt/*
var binaries embed.FS

//go:embed assets/* templates/*
var web embed.FS

type App struct {
	Path    string `json:"-"`
	Package string `json:"pkg"`
	Label   string `json:"label"`
	Enabled bool   `json:"installed"`
}

type SearchQuery struct {
	Query string `json:"query"`
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

	r := gin.Default()
	templ := template.Must(template.New("").ParseFS(web, "templates/*.html"))
	r.SetHTMLTemplate(templ)
	r.StaticFS("/public", http.FS(web))
	r.POST("/apps", func(c *gin.Context) {
		var queryApp SearchQuery
		c.BindJSON(&queryApp)

		var apps Apps

		query := strings.ToLower(queryApp.Query)
		for _, v := range pkgs {
			if strings.Contains(strings.ToLower(v.Package), query) || strings.Contains(strings.ToLower(v.Label), query) {
				apps = append(apps, v)
			}
		}
		sort.Sort(apps)
		c.JSON(200, gin.H{
			"apps": apps,
		})
	})
	r.POST("/do", func(c *gin.Context) {
		var queryApp App
		c.BindJSON(&queryApp)
		toggleApp(pkgs[queryApp.Package])
		c.JSON(200, gin.H{"status": "lol wtf"})
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	r.Run()
}

func worker(appChan, workChan chan *App) {
	for app := range workChan {
		out, err := device.RunCommand(fmt.Sprintf("%s d badging %s", aapt, app.Path))
		if err != nil {
			log.Printf("failed to retrieve package label for apk at path %s: %q", app.Path, err)
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
			log.Printf("Failed to run uninstall command on %s: %q\n", app.String(), err)
			return
		}
		if !strings.Contains(out, "Success") {
			log.Printf("Failed to uninstall %s\n", app.String())
			return
		}
		log.Printf("Successfully uninstalled %s\n", app.String())
		app.Enabled = false
		return
	}
	out, err := device.RunCommand(fmt.Sprintf("pm install-existing %s", app.Package))
	if err != nil {
		log.Printf("Failed to run reinstall command on %s: %q\n", app.String(), err)
		return
	}
	if !strings.Contains(out, "Success") {
		log.Printf("Failed to reinstall %s\n", app.String())
		return
	}
	log.Printf("Successfully reinstalled %s\n", app.String())
	app.Enabled = true
}
