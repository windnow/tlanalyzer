package myfsm

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (m *Monitor) ScanFile(filePath string) ([]Event, error) {

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))
	reNewEvent := regexp.MustCompile(`^\d\d:\d\d\.\d{6}`)

	fsm := &myFSM{fileName: fileNameWithoutExt}
	file, scanner, err := getFileScanner(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		file.Close()
	}()

	for scanner.Scan() {
		select {
		case <-m.ctx.Done():
			errMsg := fmt.Sprintf("СКАНИРОВАНИЕ ФАЙЛА \"%s\" ПРЕРВАНО (ПО СИГНАЛУ)", fileName)
			return nil, errors.New(errMsg)
		default:
			processLine(fsm, scanner.Text(), reNewEvent)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	return fsm.events, nil
}

func processLine(fsm *myFSM, line string, reNewEvent *regexp.Regexp) {
	line = strings.TrimSpace(line)
	if reNewEvent.MatchString(line) {
		if fsm.Event != nil {
			fsm.Event = fsm.FinalizeEvent
		} else {
			fsm.Event = fsm.NewEvent
		}
	} else {
		line = "\n" + line
	}

	for _, c := range line {
		fsm.Update(c)
	}

}

func getFileScanner(filePath string) (file *os.File, scanner *bufio.Scanner, err error) {
	file, err = os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}

	scanner = bufio.NewScanner(file)
	scanner.Split(scanLinesWithoutBOM)

	scanner.Buffer(make([]byte, 0), 1024*1024)
	return
}
