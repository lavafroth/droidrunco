package bridge

import (
	"log"
	// "time"

	"github.com/lavafroth/droidrunco/meta"
	adb "github.com/zach-klippenstein/goadb"
)

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
