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
			e, err := ScanFile(path, 0)
			if err != nil {
				info = fmt.Sprintf("%s ERROR: \"%s\".\n\tПовторное чтение (увеличим размер буфера чтения)", info, err.Error())
				e, err = ScanFile(path, 512)
				if err != nil {
					duration := time.Since(start)
					info = fmt.Sprintf("%s. Ошибка с множителем 512. Длительность выполнения: %d с.", info, duration/time.Second)
					return nil //err
				}
			}
			duration := time.Since(start)
			l := len(e)
			if l > 0 {
				log.Printf("%s. Прочитано %d записей за (длительность: %d сек.)", info, l, duration/time.Second)
			}
			events = append(events, e...)

		}

		return nil
	})

	return events

}
