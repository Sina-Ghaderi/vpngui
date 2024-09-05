package handler

import (
	"context"
	"snixconnect/internal/gui"
	"snixconnect/internal/logs"
)

type gList struct {
	Name, FriendlyName string
}

func runSnixConnect(appdir string) {

	app := gui.NewGuiHandler(appdir)
	logger := logs.NewLogger("[NET]", app.GuiLogHandler())

	authProvider := func(g []gList, banner string) bool {
		guiGroup := make([]gui.GroupSelect, len(g))
		for _, v := range g {
			gs := gui.GroupSelect{Name: v.Name, FriendlyName: v.FriendlyName}
			guiGroup = append(guiGroup, gs)
		}

		_, ok := app.UserCerdential(guiGroup, banner)
		return ok

	}

	_ = authProvider

	connHandler := func(ctx context.Context, addr string) {
		config, guid := app.GetAppConfig(), app.GetTunnelGUID()
		_, _ = config, guid

		statuschan, errchan := simVpn(ctx)

		var stillReconnecting bool

		for {
			select {
			case err := <-errchan:
				logger.Print(err)
				app.SetConnStatus(gui.NewStatusDisconnected(gui.FlagConnFailed))
				return

			case status := <-statuschan:
				switch status {
				case 1:
					app.SetConnStatus(gui.NewStatusConnected(gui.ConnectionStats{}))
					stillReconnecting = false

				case 0:
					flag := gui.FlagConnecting
					app.SetConnStatus(gui.NewStatusConnecting(flag))
					stillReconnecting = false

				case 3:
					if !stillReconnecting {
						flag := gui.FlagReconnecting
						app.SetConnStatus(gui.NewStatusConnecting(flag))
						stillReconnecting = true
					}
				}
			}
		}
	}

	app.SetConnectHandler(connHandler)
	showErrMsg(app.RenderWindow())
}

// simulate vpn library:
func simVpn(ctx context.Context) (chan int, chan error) {
	err := make(chan error)
	status := make(chan int)

	go func() {
		status <- 0
		<-ctx.Done()
		close(err)
	}()

	return status, err

}
