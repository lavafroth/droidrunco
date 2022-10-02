package bridge

import (
	"fmt"
	"log"
	"strings"
	"time"
	"github.com/lavafroth/droidrunco/app"
	"embed"
)

var work = make(chan *app.App, 8)

//go:embed extractor/build/*
var binaries embed.FS

func labelWorker() {
	for app := range work {
		label, err := device.RunCommand(fmt.Sprintf("%s %s", extractor, app.Path))
		if err != nil {
			log.Printf("Failed to retrieve package label: %q, path: %s", err, app.Path)
		}

		app.SetLabel(strings.Trim(label, "\n"))

		if k := db.Get(app.Id); k != nil {
			app.Meta = *k
		}

		if app.Description == "" {
			app.Description = "Description not yet available."
		}
	}
}

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
