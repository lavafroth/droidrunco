package bridge

import (
	"github.com/lavafroth/droidrunco/meta"
	adb "github.com/zach-klippenstein/goadb"

	_ "embed"
	"fmt"
	"log"
	"time"
)

// Thank you Irfan Latif
// https://android.stackexchange.com/questions/90141/obtain-package-name-and-common-name-of-apps-via-adb

//go:embed extractor.dex
var extractorDex []byte

var device *adb.Device
var db meta.DB
var client *adb.Adb

func Init() {
	var err error

	db, err = meta.Init()
	if err != nil {
		log.Fatal(err)
	}

	client, err = adb.New()
	if err != nil {
		log.Fatalf("failed to start adb server: %q", err)
	}
	client.StartServer()
	device = client.Device(adb.AnyDevice())
	if err := push("/data/local/tmp/extractor.dex"); err != nil {
		log.Fatal(err)
	}

	_, err = device.RunCommand("CLASSPATH=/data/local/tmp/extractor.dex app_process / Main")
	if err != nil {
		log.Fatalf("failed to execute extractor dalvik executable: %q", err)
	}
}

func Close() {
	client.KillServer()
}

func push(remote string) error {
	remoteHandle, err := device.OpenWrite(remote, 0o755, time.Now())
	if err != nil {
		return fmt.Errorf("failed to open handle with write permissions on file %s: %q", remote, err)
	}
	defer remoteHandle.Close()
	if _, err := remoteHandle.Write(extractorDex); err != nil {
		return fmt.Errorf("failed to copy data from local file handle to remote file: %q", err)
	}
	return nil
}
