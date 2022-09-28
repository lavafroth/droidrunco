package bridge

import (
	"fmt"
	"log"
	"strings"

	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/meta"
)

var Cache app.Apps
var Updated bool

func Refresh() {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		log.Fatalf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	var fresh app.Apps
	gotFreshPackages := false

	for _, line := range strings.Split(out, "\n") {
		_, line, _ := strings.Cut(line, ":")
		i := strings.LastIndex(line, "=")
		path, pkg := line[:i], line[i+1:]

		// If we can already find the same package in the old list,
		// we don't bother looking up its label name.
		App := Cache.Get(pkg)
		if App == nil {
			gotFreshPackages = true
			App = &app.App{Meta: meta.Meta{Package: pkg}, Path: path, Enabled: true}
			work <- App
		}
		fresh = append(fresh, App)
	}

	if !gotFreshPackages {
		return
	}

	Updated = gotFreshPackages

	for _, app := range Cache {
		// The app was previously enabled
		// but is no more in the new list.
		if fresh.Get(app.Package) == nil {
			// We can conclude that the
			// app has been disabled.
			app.Enabled = false
			fresh = append(fresh, app)
		}
	}
	Cache = fresh
}

func Toggle(App *app.App) string {
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
