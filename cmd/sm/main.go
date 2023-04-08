package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/windnow/sm/internal/myfsm"
)

func main() {

	if len(os.Args) == 1 {
		log.Fatal("directory name not specified")
	}
	rootDir := os.Args[1]
	mask := "*.log"
	var events []*myfsm.Event

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		matched, err := filepath.Match(mask, info.Name())
		if err != nil {
			return err
		}

		if !info.IsDir() && matched {
			fmt.Print("Чтение файла " + path)
			e, err := myfsm.ScanFile(path)
			if err != nil {
				fmt.Println(" ERROR: " + err.Error())
				return nil //err
			}
			fmt.Println()
			events = append(events, e...)

		}

		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}

	if len(events) > 0 {
		fmt.Println(events[len(events)-1].Fields["Sql"])
	}

}
