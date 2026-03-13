package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Printf("ted: opening %s\n", os.Args[1])
	} else {
		fmt.Println("ted: no file specified")
	}
}
