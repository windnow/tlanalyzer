package myfsm

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"time"
)

func WalkDir(rootDir string) []*Event {

	mask := "*.log"
	var events []*Event

	filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		matched, err := filepath.Match(mask, info.Name())
		if err != nil {
			return err
		}

		if !info.IsDir() && matched {
			info := fmt.Sprintf("Чтение %s", path)
			start := time.Now()
			e, err := ScanFile(path)
			if err != nil {
				duration := time.Since(start)
				log.Printf("%s. Ошибка. %s. Длительность: %d мк.с.", info, err.Error(), duration/time.Microsecond)
				return nil //err
			}
			duration := time.Since(start)
			l := len(e)
			if l > 0 {
				log.Printf("%s. Прочитано %d записей за (длительность: %d мк.с.)", info, l, duration/time.Microsecond)
			}
			events = append(events, e...)

		}

		return nil
	})

	return events

}
