package myfsm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func WalkDir(rootDir string) []Event {

	mask := "*.log"
	var events []Event

	log := &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%]: %time% - %msg%\n",
		},
	} //logrus.New()
	filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		matched, err := filepath.Match(mask, info.Name())
		if err != nil {
			return err
		}

		if !info.IsDir() && matched {
			info := fmt.Sprintf("`%s`", path)
			start := time.Now()
			e, err := ScanFile(path)
			if err != nil {
				duration := time.Since(start)
				log.Errorf("%s: Ошибка. %s. Длительность: %d мк.с.", info, err.Error(), duration/time.Microsecond)
				return nil //err
			}
			duration := time.Since(start)
			l := len(e)
			if l > 0 {
				log.Infof("%s: %d, %d, %d", info, l, duration/time.Microsecond, (duration/time.Microsecond)/time.Duration(l))
			}
			events = append(events, e...)

		}

		return nil
	})

	return events

}
