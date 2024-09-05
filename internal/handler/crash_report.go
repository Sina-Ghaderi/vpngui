package handler

import (
	"fmt"
	"os"
	"snixconnect/internal/gui"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

func crashReportDump(logFile *os.File) error {
	err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(logFile.Fd()))
	if err != nil {
		return err
	}
	os.Stderr = logFile
	return nil
}
func showErrMsg(err error) {
	if err == nil {
		return
	}
	gui.WinErrorBox(err)
}

func showPanicErr(path string) {
	if err := recover(); err != nil {
		show := fmt.Errorf("SnixConnect program crashed. \n\nYou can find the crash report file at path: %s", path)
		showErrMsg(show)
		win.ShellExecute(win.HWND_DESKTOP, nil, windows.StringToUTF16Ptr(path), nil, nil, win.SW_SHOWNORMAL)
		panic(err)
	}
}

func RunSnixConnectApp() {

	var appDir string
	if len(os.Args) > 0 {
		appDir = os.Args[0]
	}

	file, path, err := gui.CrashReportFile(appDir)
	if err != nil {
		showErrMsg(err)
		return
	}

	err = crashReportDump(file)
	if err != nil {
		showErrMsg(fmt.Errorf("error setting crash log file handler\n\n%v", err))
		return
	}

	defer showPanicErr(path)
	if err := simCheckDriver(); err != nil {
		showErrMsg(fmt.Errorf("error loading driver, reinstalling the program may solve the issue\n\n%v", err))
		return
	}

	runSnixConnect(appDir)
}

// Simulate check driver version:
func simCheckDriver() error {
	return nil
}
