package gui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"snixconnect/internal/bsync"
	"snixconnect/internal/logs"
	"sync"
	"sync/atomic"
	"time"

	"snixconnect/pkg/walk"

	"golang.org/x/sys/windows"
)

const globalAppMutex = "Global\\SnixConnectVPNClientGui"

type ConnectHandler func(context.Context, string)
type appGuiHandler struct {
	mainProperty   *winMainProperty
	logsProPerty   *winLogsProperty
	optionProperty *winOptionProperty
	credProperty   *winCredProperty
	bannerProperty *winBannerProperty
	aboutProperty  *winAboutProperty
	handler        *connHandler
	tundeviceGUID  *windows.GUID
	closeWaitGroup sync.WaitGroup
}

type connHandler struct {
	connectFunc     ConnectHandler
	connHandler     atomic.Value
	ctxCancelFunc   context.CancelFunc
	handlerIsCancel bool
}

func NewGuiHandler(localAppDir string) *appGuiHandler {
	app := &appGuiHandler{
		logsProPerty: new(winLogsProperty), mainProperty: new(winMainProperty),
		handler: &connHandler{connectFunc: func(ctx context.Context, s string) {}},
	}

	localAppDirByCmd = localAppDir
	app.optionProperty = new(winOptionProperty)
	app.credProperty = &winCredProperty{c: new(userCredential)}
	app.logsProPerty.logModel = newAppLogViewModel()
	lfnoop := func(s string) { app.logsProPerty.logModel.addLogToItems(s) }
	app.logsProPerty.updateLogTable.Store(lfnoop)
	app.logsProPerty.connStats = new(ConnectionStats)
	app.logsProPerty.updateDetails.Store(func() {})
	app.bannerProperty = new(winBannerProperty)
	app.aboutProperty = new(winAboutProperty)
	if app.handler.connectFunc == nil {
		app.handler.connectFunc = func(context.Context, string) {}
	}
	app.mainProperty.tray = &appTrayNotify{windowsIsCore: winOSIsCore()}
	logger = logs.NewLogger("[GUI]", app.GuiLogHandler())
	return app
}

func (g *appGuiHandler) SetConnectHandler(f ConnectHandler) { g.handler.connectFunc = f }

func (g *appGuiHandler) alreadyRunning() error {
	err := createWin32Mutex(globalAppMutex)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		bringProcWinUp(mainWinName)
		os.Exit(0)
	}
	if err != nil {
		err = fmt.Errorf("create win32 mutex error: %v", err)
	}

	return err
}

func (g *appGuiHandler) renderMainWindow() error {
	runtime.LockOSThread()
	if err := g.alreadyRunning(); err != nil {
		return err
	}
	windows.SetProcessPriorityBoost(windows.CurrentProcess(), false)
	err := g.mainProperty.newMainWindow().Create()
	if err != nil {
		return err
	}
	var sid uint32
	pid := os.Getpid()
	windows.ProcessIdToSessionId(uint32(pid), &sid)
	vrs := fmt.Sprintf("%s %s", snixConnectVersion, runtime.GOARCH)
	logger.Printf("starting SnixConnect v%s with PID %d on %s", vrs, pid, osRuningInfo())
	logger.Printf("starting UI process for user '%s' in session %d", currentWinUser(), sid)
	g.mainProperty.setAppGuiTweaks()
	g.mainProperty.newTrayIcon()
	g.mainProperty.tray.attachExitAction(func() { go g.exitSnixConnect() })

	binder, err := loadUserCerdential()
	if err != nil {
		binder = new(userCredential)
	}
	config, err := loadUserAppConfig()
	if err != nil {
		config = new(UserAppConfig)
		config.CredentialCache = true
	}

	g.tundeviceGUID, err = getTunGuidValue()
	if err != nil {
		logger.Print(err)
	}

	g.optionProperty.currentConfig = config
	g.credProperty.c = binder
	g.mainProperty.serverLineEdit.SetText(binder.ServerAddress)

	// About button handler:
	viewAboutHandler := func() { g.aboutProperty.newAboutDialog() }
	g.mainProperty.aboutButton.Triggered().Attach(viewAboutHandler)
	g.mainProperty.tray.attachAbotAction(viewAboutHandler)

	// Log button handler:
	viewLogHandler := func() { g.logsProPerty.newViewLogsDialog() }
	g.mainProperty.viewLogButton.Clicked().Attach(viewLogHandler)
	onButtonPressEnter(g.mainProperty.viewLogButton.KeyUp(), viewLogHandler)

	// Setting button handler:
	optWinHandler := func() { g.optionProperty.newSettingDialog() }
	g.mainProperty.settingsButton.Triggered().Attach(optWinHandler)

	// Connect/Disconnect handler
	g.mainProperty.connectButton.Clicked().Attach(g.exeConnHandler)
	onButtonPressEnter(g.mainProperty.connectButton.KeyUp(), g.exeConnHandler)
	g.mainProperty.tray.attachConnectAction(g.exeConnHandler)
	g.SetConnStatus(NewStatusDisconnected(FlagDisconnected))

	g.mainProperty.mainWindow.Closing().Attach(g.handleCloseToTry)
	g.mainProperty.mainWindow.Run()
	return nil
}

func (g *appGuiHandler) setButtonDisconnect() {
	g.handler.handlerIsCancel = true
	g.mainProperty.connectButton.SetText("Disconnect")
	g.mainProperty.tray.connectAction.SetText("Disconnect")
	disconnect := func() {
		logger.Print("user has requested to disconnect from vpn server")
		g.runDisconnectFunc()
	}
	g.handler.connHandler.Store(walk.EventHandler(bsync.OnceFunc(disconnect)))
}

func (g *appGuiHandler) setButtonConnect() {
	g.handler.handlerIsCancel = false
	g.mainProperty.connectButton.SetText("Connect")
	g.mainProperty.tray.connectAction.SetText("Connect")
	connectFunc := func() { go g.runConnectFunc() }
	g.handler.connHandler.Store(walk.EventHandler(bsync.OnceFunc(connectFunc)))
}

func (g *appGuiHandler) runConnectFunc() {

	g.closeWaitGroup.Add(1)
	defer g.closeWaitGroup.Done()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	g.mainProperty.serverLineEdit.SetEnabled(false)
	defer g.mainProperty.serverLineEdit.SetEnabled(true)
	var ctx context.Context
	ctx, g.handler.ctxCancelFunc = context.WithCancel(context.Background())
	defer g.handler.ctxCancelFunc()
	logger.Print("user has requested to connect to the vpn server")
	addr := g.mainProperty.serverLineEdit.Text()
	g.handler.connectFunc(ctx, addr)
}

func (g *appGuiHandler) handleCloseToTry(canceled *bool, reason walk.CloseReason) {
	*canceled = true
	g.mainProperty.hideWindow()
}

func (g *appGuiHandler) exitSnixConnect() {

	code := 1
	const waitForThings = 3 * time.Second
	cancelCtx := func(cancelFunc context.CancelFunc) {
		g.closeWaitGroup.Wait()
		code = 0
		cancelFunc()
	}

	ctx, cancel := context.WithTimeout(context.Background(), waitForThings)
	if g.handler.handlerIsCancel {
		g.handler.connHandler.Load().(walk.EventHandler)()
		go cancelCtx(cancel)
		<-ctx.Done()
	} else {
		code = 0
	}

	g.mainProperty.tray.trayIcon.Dispose()
	os.Exit(code)
}

func (g *appGuiHandler) runDisconnectFunc() {
	if g.handler.ctxCancelFunc == nil {
		return
	}
	g.handler.ctxCancelFunc()
}

func (g *appGuiHandler) exeConnHandler() {
	if g.handler.handlerIsCancel {
		g.handler.connHandler.Load().(walk.EventHandler)()
		return
	}
	if !g.mainProperty.validateAddress() {
		return
	}
	g.handler.connHandler.Load().(walk.EventHandler)()
}

func (g *appGuiHandler) RenderWindow() error { return g.renderMainWindow() }

func (g *appGuiHandler) GuiLogHandler() func(string)  { return g.logsProPerty.loggerFunc }
func (g *appGuiHandler) GetTunnelGUID() *windows.GUID { return g.tundeviceGUID }

func (g *appGuiHandler) GetAppConfig() *UserAppConfig {
	g.optionProperty.mutex.Lock()
	defer g.optionProperty.mutex.Unlock()
	config := *g.optionProperty.currentConfig
	return &config
}

func (g *appGuiHandler) UserCerdential(groups []GroupSelect, banner string) (*UserCredential, bool) {
	c := new(UserCredential)
	urladdr, err := parseRawURL(g.mainProperty.serverLineEdit.Text())
	if err != nil {
		urladdr = new(url.URL)
	}
	cache := g.optionProperty.currentConfig.CredentialCache
	cache = cache && urladdr.String() == g.credProperty.c.ServerAddress
	cache = cache && len(urladdr.String()) > 0
	cache = cache && len(g.credProperty.c.Username) != 0
	cache = cache && len(g.credProperty.c.Password) != 0
	groupExist := false
	for _, v := range groups {
		if v.Name == g.credProperty.c.Group && len(v.Name) != 0 {
			groupExist = true
			break
		}
	}
	groupExist = groupExist || len(groups) == 0

	if g.credProperty.c.LastConnected && cache && groupExist {
		c.Username = g.credProperty.c.Username
		c.Password = g.credProperty.c.Password
		if len(groups) != 0 {
			c.Group = g.credProperty.c.Group
		}
		logger.Printf("using previously cached credentials for host %s", urladdr.Host)
		return c, true
	}

	logger.Printf("prompt user credential dialog for host %s", urladdr.Host)
	if !cache {
		g.credProperty.c.UserCredential = UserCredential{}
	}

	dlgchan := make(chan int)
	g.credProperty.newCredentialDialog(dlgchan, groups, banner)
	switch <-dlgchan {
	case walk.DlgCmdOK:
	case walk.DlgCmdAbort:
		logger.Print("credential dialog terminated by another thread")
		return c, false
	default:
		logger.Print("user declined to provide vpn connection credentials")
		return c, false
	}

	c.Username = g.credProperty.c.Username
	c.Password = g.credProperty.c.Password
	c.Group = g.credProperty.c.Group
	return c, true
}

func (g *appGuiHandler) ShowServerBanner(banner string) {
	urladdr, err := parseRawURL(g.mainProperty.serverLineEdit.Text())
	if err != nil {
		urladdr = new(url.URL)
	}
	logger.Printf("prompt vpn login banner dialog for host %s", urladdr.Host)
	g.bannerProperty.newBannerDialog(banner)
}
