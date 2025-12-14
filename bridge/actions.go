package bridge

import (
	"fmt"
	"strings"

	"github.com/lavafroth/droidrunco/app"
	"github.com/lavafroth/droidrunco/meta"
)

var Cache app.Apps

// Refresh returns true if the refresh resulted in a changed set of packages
func Refresh() (bool, error) {
	out, err := device.RunCommand("pm list packages -f")
	if err != nil {
		return false, fmt.Errorf("failed to fetch list of packages: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	var fresh app.Apps = map[string]*app.App{}
	changed := false

	for _, line := range strings.Split(out, "\n") {
		_, line, _ := strings.Cut(line, ":")
		i := strings.LastIndex(line, "=")
		path, id := line[:i], line[i+1:]

		// If we can already find the same package in the old list,
		// we don't bother looking up its label name.
		App, ok := Cache[id]
		if !ok {
			changed = true

			metadata := meta.Meta{}

			if metadataPtr := db.Get(id); metadataPtr != nil {
				metadata = *metadataPtr
			}

			App = &app.App{Id: id, Meta: metadata, Path: path, Enabled: true}
		}
		fresh[id] = App
	}

	// has any app been disabled? Present in old map, absent in new.
	for id, app := range Cache {
		if _, inNewMap := fresh[id]; !inNewMap {
			changed = true
			app.Enabled = false
			fresh[id] = app
		}
	}

	if !changed {
		return false, nil
	}

	out, err = device.RunCommand("CLASSPATH=/data/local/tmp/extractor.dex app_process / Main")
	if err != nil {
		return false, fmt.Errorf("failed to extract package labels: %q", err)
	}
	out = strings.Trim(out, "\n\t ")
	for _, line := range strings.Split(out, "\n") {
		_, idAndLabel, _ := strings.Cut(line, " ")
		id, label, _ := strings.Cut(idAndLabel, " ")

		App, ok := fresh[id]
		if ok && App.Label == "" {
			App.SetLabel(label)
		}
	}

	Cache = fresh
	return true, nil
}

func Toggle(App *app.App) string {
	if App.Enabled {
		// Isssue the uninstall command for the respective package
		out, err := device.RunCommand(fmt.Sprintf("pm uninstall -k --user 0 %s", App.Id))
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
	out, err := device.RunCommand(fmt.Sprintf("pm dump %s", App.Id))
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
