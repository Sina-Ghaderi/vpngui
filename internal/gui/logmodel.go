package gui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"snixconnect/pkg/walk"
)

const maxLogLinesDisplayed = 10000

type appLogViewModel struct {
	walk.ReflectTableModelBase
	items      []appLogEntry
	muxLogData sync.Mutex
}

type appLogEntry struct {
	Stamp time.Time
	Line  string
}

func (m *appLogViewModel) Items() interface{} {
	m.muxLogData.Lock()
	defer m.muxLogData.Unlock()
	return m.items
}

func newAppLogViewModel() *appLogViewModel {
	return &appLogViewModel{items: make([]appLogEntry, 0)}
}

func (m *appLogViewModel) addLogToItems(logline string) bool {
	m.muxLogData.Lock()
	defer m.muxLogData.Unlock()
	m.items = append(
		m.items, appLogEntry{Line: logline, Stamp: time.Now()},
	)

	var updateAllRows bool
	if len(m.items) > maxLogLinesDisplayed {
		m.items = m.items[len(m.items)-maxLogLinesDisplayed:]
		updateAllRows = true
	}
	return updateAllRows
}

func (m *appLogViewModel) saveLogsToFile(path string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: saving logs to file: %v", err)
		}
	}()

	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	f, err := os.OpenFile(path, flag, filePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	m.muxLogData.Lock()
	defer m.muxLogData.Unlock()

	for i := range m.items {
		stamp := m.items[i].Stamp.Format("2006-01-02 15:04:05.000: ")
		if _, err := f.Write([]byte(stamp)); err != nil {
			return err
		}

		line := m.items[i].Line + "\n"
		if _, err := f.Write([]byte(line)); err != nil {
			return err
		}
	}
	return nil

}

func (p *appLogViewModel) StyleCell(style *walk.CellStyle) {
	i := style.Row()
	if len(p.items) <= i || i < 0 {
		return
	}

	m := p.items[i]
	showInRed := strings.Contains(m.Line, "Error:") ||
		strings.Contains(m.Line, "Fatal:")

	if showInRed {
		style.TextColor = colorFailed
		return
	}

	warning := strings.Contains(m.Line, "Warning:")
	if warning {
		style.TextColor = walk.Color(0x0c90c2)
		return
	}

	if strings.Contains(m.Line, "link is up") {
		style.TextColor = colorConnect
		return
	}
}
