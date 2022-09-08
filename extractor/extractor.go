package main

import (
	"github.com/shogo82148/androidbinary/apk"
	"github.com/shogo82148/androidbinary"
	"os"
	"fmt"
)

func main() {
	if len(os.Args) != 2 {
		return
	}
	pkg, _ := apk.OpenFile(os.Args[1])
	defer pkg.Close()

	resConfigEN := &androidbinary.ResTableConfig{
		Language: [2]uint8{uint8('e'), uint8('n')},
	}
	appLabel, _ := pkg.Label(resConfigEN) // get app label for en translation
	fmt.Println(appLabel)
}
