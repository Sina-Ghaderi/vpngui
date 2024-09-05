package logs

import (
	"log"
	"strings"
)

type guiLogger struct{ guiLogWriter func(string) }

func NewLogger(prefix string, logger func(string)) *log.Logger {
	out := new(guiLogger)
	out.guiLogWriter = logger
	l := log.New(out, prefix+" ", 0)
	return l
}

func (l *guiLogger) Write(b []byte) (int, error) {
	sform := strings.SplitAfterN(string(b), " ", 2)
	if len(sform) == 2 {
		if len(sform[1]) > 0 {
			sform[1] = strings.ToUpper(sform[1][0:1]) + sform[1][1:]
		}
	}

	l.guiLogWriter(strings.Trim(strings.Join(sform, ""), "\n"))
	return len(b), nil
}
