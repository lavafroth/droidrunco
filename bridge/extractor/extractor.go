package main

import (
	"fmt"
	"github.com/shogo82148/androidbinary"
	"github.com/shogo82148/androidbinary/apk"
	"os"
)

func getLabel(path string) (string, error) {

	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if (fileInfo.Size() >> 20) > 400 {
		return "", fmt.Errorf("file to be parsed is larger than 400MB")
	}

	pkg, err := apk.OpenFile(os.Args[1])

	if err != nil {
		return "", err
	}
	defer pkg.Close()

	resConfigEN := &androidbinary.ResTableConfig{
		Language: [2]uint8{uint8('e'), uint8('n')},
	}
	appLabel, err := pkg.Label(resConfigEN) // get app label for en translation
	if err != nil {
		return "", err
	}

	return appLabel, nil
}

func main() {
	if len(os.Args) != 2 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("")
		}
	}()

	path := os.Args[1]
	label, _ := getLabel(path)

	// if err != nil {
	// fmt.Fprintf(os.Stderr, "%v", err)
	// }

	fmt.Println(label)
}
