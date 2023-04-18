package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func main() {
	begin := time.Now()
	if len(os.Args) == 1 {
		log.Fatal("directory name not specified")
	}
	rootDir := os.Args[1]

	myfsm.ProcessLogs(rootDir, func(events []myfsm.Event) {
		fmt.Println("ВСЕГО ПРОЧИТАНО", len(events))
	})

	log.Printf("Общее время выполнения: %d", time.Since(begin)/time.Second)

}
