package gui

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"snixconnect/pkg/walk"
	"sync"
	"time"

	"github.com/lxn/win"
)

const (
	FlagConnected StatusFlag = iota
	FlagConnecting
	FlagReconnecting
	FlagAuthFailed
	FlagRejected
	FlagConnFailed
	FlagDisconnected
)

type trayTextTitlePath struct{ message, title, iconpath string }

var trayMessageText = [...]trayTextTitlePath{
	FlagConnected: {
		message:  "Connection to the VPN server was successful; link is up.",
		title:    "Connected",
		iconpath: connIconConnected,
	},
	FlagReconnecting: {
		iconpath: connIconReconnect,
		message:  "The VPN connection has been lost, attempting to reconnect to the server...",
		title:    "Reconnecting...",
	},

	FlagAuthFailed: {
		iconpath: connIconFailed,
		message:  "The user's credentials were not accepted by the server; authentication failed.",
		title:    textAuthFailed,
	},

	FlagConnFailed: {
		title:    textConnFailed,
		message:  "Connection to the VPN server failed; connection could not be established.",
		iconpath: connIconFailed,
	},

	FlagRejected: {
		title:    textRejected,
		message:  "The user session was actively rejected by the server; connection failed.",
		iconpath: connIconFailed,
	},

	FlagDisconnected: {
		title:    textDisconnected,
		message:  "Disconnected from the VPN server; link is down.",
		iconpath: connIconDisconnected,
	},
}

type StatusFlag byte

var currentStatus statusIndicator
var statusMutex sync.Mutex

type statusConnecting struct{ statusFlag StatusFlag }
type statusDisconnected struct{ statusFlag StatusFlag }
type statusConnected struct {
	connInfo *ConnectionStats
	cancel   context.CancelFunc
	finished chan struct{}
}

type statusIndicator interface {
	applyStatus(*appGuiHandler)
	restoreChanges(*appGuiHandler)
}

func (g *appGuiHandler) SetConnStatus(status statusIndicator) {
	if status == nil {
		return
	}

	statusMutex.Lock()
	defer statusMutex.Unlock()

	if currentStatus != nil {
		currentStatus.restoreChanges(g)
	}

	status.applyStatus(g)
	currentStatus = status
}

func NewStatusConnected(s ConnectionStats) *statusConnected {
	return &statusConnected{connInfo: &s}
}

func NewStatusDisconnected(flag StatusFlag) *statusDisconnected {
	return &statusDisconnected{statusFlag: flag}
}

func NewStatusConnecting(flag StatusFlag) *statusConnecting {
	return &statusConnecting{statusFlag: flag}
}

func (g *appGuiHandler) connectedRoutine(ctx context.Context, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	info := g.logsProPerty.connStats
	if info.RX == nil {
		info.RX = func() uint64 { return 0 }
	}
	if info.TX == nil {
		info.TX = func() uint64 { return 0 }
	}

	if info.ConnectedSince == (time.Time{}) {
		info.ConnectedSince = time.Now()
	}

	for {
		select {
		case <-ticker.C:
			g.mainProperty.connRxTxLable[0].SetText(formatTransceive(info.RX()))
			g.mainProperty.connRxTxLable[1].SetText(formatTransceive(info.TX()))
			g.mainProperty.connRxTxLable[2].SetText(formatTimeText(info.ConnectedSince))

		case <-ctx.Done():
			return
		}
	}

}

func (s *statusConnected) applyStatus(g *appGuiHandler) {
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())
	s.finished = make(chan struct{})
	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)

	g.mainProperty.tray.statusAction.SetText(trayConnected)
	g.mainProperty.tray.trayIcon.SetToolTip(trayToolTipText(trayConnected))
	g.mainProperty.connStatusMsg.SetTextColor(colorConnect)

	g.mainProperty.connStatusMsg.SetText(textConnected)
	g.showTrayNotifyMsg(FlagConnected)
	g.drawTrayIconStatus(FlagConnected)

	g.mainProperty.connRxTxLable[0].SetText("0B")
	g.mainProperty.connRxTxLable[1].SetText("0B")
	g.mainProperty.connRxTxLable[2].SetText("0s")
	g.mainProperty.connRxTxLable[0].SetEnabled(true)
	g.mainProperty.connRxTxLable[1].SetEnabled(true)
	g.mainProperty.connRxTxLable[2].SetEnabled(true)

	g.mainProperty.connStatusBox.SetVisible(true)
	g.logsProPerty.connStats = s.connInfo
	g.logsProPerty.detailsUpdater()
	go g.connectedRoutine(ctx, s.finished)

	urladdr, err := parseRawURL(g.mainProperty.serverLineEdit.Text())
	if err != nil {
		urladdr = new(url.URL)
	}
	g.setButtonDisconnect()

	lconn := urladdr.String() == g.credProperty.c.ServerAddress
	cache := g.optionProperty.currentConfig.CredentialCache
	if lconn && g.credProperty.c.LastConnected == cache {
		return
	}
	g.credProperty.c.LastConnected = cache
	g.credProperty.c.ServerAddress = urladdr.String()
	err = saveUserCredential(g.credProperty.c)
	if err != nil {
		go winErrorBox(g.mainProperty.mainWindow, err)
	}
}

func (s *statusConnecting) applyStatus(g *appGuiHandler) {
	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)

	g.drawTrayIconStatus(s.statusFlag)

	switch s.statusFlag {
	case FlagReconnecting:
		g.mainProperty.tray.statusAction.SetText(trayReconnecting)
		g.mainProperty.tray.trayIcon.SetToolTip(trayToolTipText(trayReconnecting))
	default:
		g.mainProperty.tray.statusAction.SetText(trayConnecting)
		g.mainProperty.tray.trayIcon.SetToolTip(trayToolTipText(trayConnecting))
	}

	g.showTrayNotifyMsg(s.statusFlag)
	g.mainProperty.connRxTxLable[0].SetEnabled(false)
	g.mainProperty.connRxTxLable[1].SetEnabled(false)
	g.mainProperty.connRxTxLable[2].SetEnabled(false)
	g.mainProperty.connProgress.SetVisible(true)
	g.setButtonDisconnect()

}
func (s *statusDisconnected) applyStatus(g *appGuiHandler) {
	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)

	trayStatusText := trayDisconnected
	colorForDisconnect := colorDisconnect
	statusText := textDisconnected
	noResetCache := true

	switch s.statusFlag {
	case FlagAuthFailed:
		trayStatusText = trayConnFailed
		colorForDisconnect = colorFailed
		statusText = textAuthFailed
		noResetCache = false

	case FlagRejected:
		trayStatusText = trayConnFailed
		colorForDisconnect = colorFailed
		statusText = textRejected
		noResetCache = false

	case FlagConnFailed:
		trayStatusText = trayConnFailed
		colorForDisconnect = colorFailed
		statusText = textConnFailed
		noResetCache = false
	}
	g.mainProperty.tray.statusAction.SetText(trayStatusText)
	g.mainProperty.tray.trayIcon.SetToolTip(trayToolTipText(trayStatusText))
	g.mainProperty.connStatusMsg.SetTextColor(colorForDisconnect)
	g.mainProperty.connStatusMsg.SetText(statusText)
	g.showTrayNotifyMsg(s.statusFlag)
	g.drawTrayIconStatus(s.statusFlag)
	g.mainProperty.connRxTxLable[0].SetText("N/A")
	g.mainProperty.connRxTxLable[1].SetText("N/A")
	g.mainProperty.connRxTxLable[2].SetText("N/A")
	g.mainProperty.connRxTxLable[0].SetEnabled(false)
	g.mainProperty.connRxTxLable[1].SetEnabled(false)
	g.mainProperty.connRxTxLable[2].SetEnabled(false)
	g.mainProperty.connStatusBox.SetVisible(true)
	g.logsProPerty.connStats = new(ConnectionStats)
	g.logsProPerty.detailsUpdater()
	g.bannerProperty.closeDialog(walk.DlgCmdAbort)
	g.credProperty.closeDialog(walk.DlgCmdAbort)
	g.setButtonConnect()

	if noResetCache || !g.credProperty.c.LastConnected {
		return
	}

	g.credProperty.c.LastConnected = false
	err := saveUserCredential(g.credProperty.c)
	if err != nil {
		go winErrorBox(g.mainProperty.mainWindow, err)
	}
}

func (s *statusConnected) restoreChanges(g *appGuiHandler) {
	s.cancel()
	<-s.finished

	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)
	g.mainProperty.connStatusBox.SetVisible(false)

}

func (s *statusConnecting) restoreChanges(g *appGuiHandler) {
	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)
	g.mainProperty.connProgress.SetVisible(false)

}
func (s *statusDisconnected) restoreChanges(g *appGuiHandler) {
	g.mainProperty.mainWindow.SetSuspended(true)
	defer g.mainProperty.mainWindow.SetSuspended(false)
	g.mainProperty.connStatusBox.SetVisible(false)
}

func (g *appGuiHandler) showTrayNotifyMsg(f StatusFlag) {
	if int(f) > len(trayMessageText) {
		return
	}
	handle := g.mainProperty.mainWindow.Handle()
	if !win.IsIconic(handle) && win.IsWindowVisible(handle) {
		win.SetForegroundWindow(handle)
		return
	}

	dpi := g.mainProperty.tray.trayIcon.DPI()
	msgInfo := trayMessageText[f]
	if msgInfo.title == "" {
		return
	}

	icon := loadWalkIconByname(msgInfo.iconpath, dpi, iconSize128x128)
	if icon == nil {
		g.mainProperty.tray.trayIcon.ShowMessage(msgInfo.title, msgInfo.message)
		return
	}
	g.mainProperty.tray.trayIcon.ShowCustom(msgInfo.title, msgInfo.message, icon)
}

func (g *appGuiHandler) drawTrayIconStatus(f StatusFlag) {
	icon := loadTrayStatusIcon(f, g.mainProperty.tray.trayIcon.DPI())
	if icon == nil {
		return
	}
	g.mainProperty.tray.trayIcon.SetIcon(icon)
}

func trayToolTipText(strStatus string) string {
	return fmt.Sprintf("SnixConnect %s", strStatus)
}
