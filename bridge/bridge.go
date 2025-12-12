package bridge

import (
	"log"
	"strings"
	"time"

	"github.com/lavafroth/droidrunco/meta"
	adb "github.com/zach-klippenstein/goadb"
)

const extractor string = "/data/local/tmp/extractor"

var device *adb.Device
var db meta.DB
var client *adb.Adb

func Init() {
	var err error
	for i := 0; i < 8; i++ {
		go labelWorker()
	}

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

	// avoid text file is busy error
	time.Sleep(1)

	out, err = device.RunCommand(extractor)
	if err != nil {
		log.Fatalf("failed to execute extractor: %q", err)
	}

	if strings.Contains(out, "not executable") {
		log.Fatalf("Failed to execute extractor: %q", out)
	}
}

func Close() {
	client.KillServer()
}
