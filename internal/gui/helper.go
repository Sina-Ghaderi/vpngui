package gui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"snixconnect/pkg/walk"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	colorConnect    = walk.Color(0x00871a)
	colorDisconnect = walk.Color(0x313131)
	colorFailed     = walk.Color(0x0600D9)
)

const (
	textConnected    = "Connected"
	textDisconnected = "Disconnected"
	textConnFailed   = "Connection Failed"
	textAuthFailed   = "Authentication Failed"
	textRejected     = "Session Rejected"
)

const (
	trayConnected    = "Status: Connected"
	trayDisconnected = "Status: Disconnected"
	trayConnFailed   = "Status: Failed"
	trayConnecting   = "Status: Connecting..."
	trayReconnecting = "Status: Reconnecting..."
)

const (
	paddingX = 75
	paddingY = 110
)

var osWinCoreProducts = map[byte]struct{}{
	0x0000000c: {},
	0x0000000d: {}, 0x0000000e: {}, 0x0000001d: {},
	0x00000027: {}, 0x00000028: {}, 0x00000029: {},
	0x0000002b: {}, 0x0000002c: {}, 0x0000002d: {},
	0x0000002e: {}, 0x00000091: {}, 0x00000092: {},
	0x00000093: {}, 0x00000094: {}, 0x0000009f: {},
	0x000000a0: {}, 0x000000a8: {}, 0x0000006d: {},
	0x0000008f: {}, 0x00000090: {}, 0x000000a9: {},
}

const appFontFamily = "Consolas"

func bringProcWinUp(name string) {
	u16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}

	handle := win.FindWindow(nil, u16)
	if handle == 0 {
		msg := "another SnixConnect instance is already running, but we couldn't bring it up\n"
		msg += "Please exit from SnixConnect and try again"
		winInfoBox(nil, msg)
		return
	}
	if win.IsIconic(handle) {
		win.ShowWindow(handle, win.SW_RESTORE)
	}
	if !win.IsWindowVisible(handle) {
		win.ShowWindow(handle, win.SW_SHOW)
	}

	win.SetForegroundWindow(handle)
}

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")
)

var logger interface {
	Printf(string, ...any)
	Print(...any)
}

type positionAdjuster interface {
	Height() int
	Width() int
	SetY(int) error
	SetX(int) error
	Handle() win.HWND
}

func winSizeWithoutTaskbar() (int, int) {
	rectvalue := new(win.RECT)
	const SPI_GETWORKAREA = 0x0030
	systemParametersInfo.Call(uintptr(SPI_GETWORKAREA),
		uintptr(0),
		uintptr(unsafe.Pointer(rectvalue)), uintptr(0),
	)
	return int(rectvalue.Bottom), int(rectvalue.Right)
}

func winErrorBox(handle walk.Form, err error) {
	ferror := fmt.Sprintf("%v", err)
	if len(ferror) > 0 {
		ferror = strings.ToUpper(ferror[0:1]) + ferror[1:]
	}
	walk.MsgBox(handle, "SnixConnect Error", ferror, walk.MsgBoxIconError)
}

func createWin32Mutex(name string) error {
	u16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	_, err = windows.CreateMutex(nil, false, u16)
	if err != nil {
		return err
	}
	return nil
}

func winInfoBox(handle walk.Form, info string) {
	if len(info) > 0 {
		info = strings.ToUpper(info[0:1]) + info[1:]
	}
	walk.MsgBox(handle, "SnixConnect Information", info, walk.MsgBoxIconInformation)
}

func WinErrorBox(err error) {
	ferror := fmt.Sprintf("%v", err)
	if len(ferror) > 0 {
		ferror = strings.ToUpper(ferror[0:1]) + ferror[1:]
	}
	walk.MsgBox(nil, "SnixConnect Error", ferror, walk.MsgBoxIconError)
}

func winAdjustPosRightBottom(handle positionAdjuster) {
	windpi := int(win.GetDpiForWindow(handle.Handle()))
	formSize := walk.Size{Width: handle.Width(), Height: handle.Height()}
	dpisize := walk.SizeFrom96DPI(walk.SizeFrom96DPI(formSize, windpi), windpi)
	refactor := float64(windpi) / 96
	yScreen, xScreen := winSizeWithoutTaskbar()
	handle.SetX(int(float64(xScreen-dpisize.Width-paddingX) / refactor))
	handle.SetY(int(float64(yScreen-dpisize.Height-paddingY) / refactor))
	win.SetForegroundWindow(handle.Handle())
}

func winAdjustPosCenter(handle positionAdjuster) { winAdjustPos(handle, 2) }
func winAdjustPos(handle positionAdjuster, xy_portion float64) {
	dpi := int(win.GetDpiForWindow(handle.Handle()))
	formSize := walk.Size{Width: handle.Width(), Height: handle.Height()}
	dpisize := walk.SizeFrom96DPI(formSize, dpi)
	yScreen, xScreen := winSizeWithoutTaskbar()
	handle.SetX(int(float64(xScreen-dpisize.Width) / xy_portion))
	handle.SetY(int(float64(yScreen-dpisize.Height) / xy_portion))
	win.SetForegroundWindow(handle.Handle())
}

func onButtonPressEnter(b *walk.KeyEvent, handler walk.EventHandler) {
	b.Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			handler()
		}
	})
}

func currentWinUser() string {
	pw_name := make([]uint16, 128)
	pwname_size := uint32(len(pw_name)) - 1
	err := windows.GetUserNameEx(windows.NameSamCompatible,
		&pw_name[0], &pwname_size)
	if err != nil {
		return ""
	}

	host := windows.UTF16ToString(pw_name)
	spName := strings.Split(host, "\\")
	if len(spName) != 2 {
		return host
	}

	return spName[1] + "@" + spName[0]

}

func parseRawURL(rawurl string) (urltype *url.URL, err error) {
	u, err := url.ParseRequestURI(rawurl)
	if err == nil && len(u.Host) > 0 {
		if u.Scheme != "https" && u.Scheme != "http" {
			return u, fmt.Errorf("invalid server address %s", rawurl)
		}
		return u, err
	}

	u, err = url.ParseRequestURI("https://" + rawurl)
	if err == nil && len(u.Host) > 0 {
		return u, err
	}

	return u, fmt.Errorf("invalid server address %s", rawurl)
}

func isvalidUrl(sURL string) (turl string, err error) {
	urladdr, err := parseRawURL(sURL)
	if err != nil {
		return
	}
	return urladdr.String(), nil
}

func winOSIsCore() bool {
	versionInfo := windows.RtlGetVersion()
	if versionInfo.MajorVersion > 6 ||
		(versionInfo.MajorVersion == 6 && versionInfo.MinorVersion >= 2) {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`Software\Microsoft\Windows NT\CurrentVersion\Server\ServerLevels`,
			registry.READ,
		)
		if err != nil {
			return false
		}
		nanoServerInteger, _, err1 := k.GetIntegerValue("NanoServer")
		serverCoreInteger, _, err2 := k.GetIntegerValue("ServerCore")
		serverGuiInteger, _, err3 := k.GetIntegerValue("Server-Gui-Shell")
		nanoServer := nanoServerInteger == 1 && err1 == nil
		serverCore := serverCoreInteger == 1 && err2 == nil
		serverGui := serverGuiInteger == 1 && err3 == nil
		k.Close()
		return (nanoServer || serverCore) && !serverGui
	}

	if _, ok := osWinCoreProducts[versionInfo.ProductType]; ok {
		return ok
	}
	return false
}

func formatTimeText(t time.Time) string {
	since := time.Since(t)
	const oneDay = 24 * time.Hour

	countTimeUnit := func(unit time.Duration) uint {
		c_uint := uint(0)
		for since > unit {
			since -= unit
			c_uint++
		}
		return c_uint
	}

	days := countTimeUnit(oneDay)
	hours := countTimeUnit(time.Hour)
	minutes := countTimeUnit(time.Minute)
	seconds := countTimeUnit(time.Second)

	var format string
	var args []any

	if days != 0 {
		args = append(args, days)
		format += "%dd"
	}

	if hours != 0 {
		args = append(args, hours)
		format += "%dh"
	}

	if minutes != 0 {
		args = append(args, minutes)
		format += "%dm"
	}

	if len(args) == 0 {
		args = append(args, seconds)
		format += "%ds"
	}

	return fmt.Sprintf(format, args...)

}

func formatTransceive(rbyte uint64) string {
	unit := ""
	value := float64(rbyte)

	const (
		BYTE = 1 << (10 * iota)
		KILOBYTE
		MEGABYTE
		GIGABYTE
		TERABYTE
		PETABYTE
		EXABYTE
	)

	switch {
	case rbyte >= EXABYTE:
		unit = "E"
		value = value / EXABYTE
	case rbyte >= PETABYTE:
		unit = "P"
		value = value / PETABYTE
	case rbyte >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case rbyte >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case rbyte >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case rbyte >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case rbyte >= BYTE:
		unit = "B"
	case rbyte == 0:
		return "0B"
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".00")
	return result + unit
}
