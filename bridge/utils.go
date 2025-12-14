package bridge

import (
	_ "embed"
	"fmt"
	// "github.com/lavafroth/droidrunco/app"
	"time"
)

// Thank you Irfan Latif
// https://android.stackexchange.com/questions/90141/obtain-package-name-and-common-name-of-apps-via-adb

//go:embed extractor.dex
var extractorDex []byte

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
