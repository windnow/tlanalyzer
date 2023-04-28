package myfsm

import (
	"os"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func (m *Monitor) loggerConfig() *logrus.Logger {
	return &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%]: %time% - %msg%\n",
		},
	}
}

func (m *Monitor) getFolders() []string {
	m.mutex.Lock()
	result := append(m.folders, m.cfg_folders...)
	m.mutex.Unlock()
	return result
}
