package gui

import (
	"fmt"
	"snixconnect/pkg/walk"
)

type appTrayNotify struct {
	trayIcon      *walk.NotifyIcon
	connectAction *walk.Action
	showWinAction *walk.Action
	hideWinAction *walk.Action
	aboutAction   *walk.Action
	exitAction    *walk.Action
	statusAction  *walk.Action

	windowsIsCore bool
}

func (g *winMainProperty) newTrayIcon() {

	g.tray.trayIcon = new(walk.NotifyIcon)
	g.tray.connectAction = new(walk.Action)
	g.tray.showWinAction = new(walk.Action)
	g.tray.hideWinAction = new(walk.Action)
	g.tray.aboutAction = new(walk.Action)
	g.tray.exitAction = new(walk.Action)
	g.tray.statusAction = new(walk.Action)

	if g.tray.windowsIsCore {
		return
	}

	err := g.showTrayIcon()
	if err == nil {
		return
	}

	err = fmt.Errorf("error running showTrayIcon: %v", err)
	go winErrorBox(nil, err)

}

func (g *winMainProperty) showTrayIcon() (err error) {
	g.tray.trayIcon, err = walk.NewNotifyIcon(g.mainWindow)
	if err != nil {
		return
	}
	g.tray.trayIcon.SetVisible(true)
	clickedMouse := func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		g.showWindow()
	}

	g.tray.trayIcon.MouseDown().Attach(clickedMouse)
	g.tray.trayIcon.MessageClicked().Attach(func() { g.showWindow() })

	tryActions := make([]*walk.Action, 0)

	g.tray.exitAction = walk.NewAction()
	g.tray.exitAction.SetText("Exit")
	tryActions = append(tryActions, g.tray.exitAction)

	g.tray.aboutAction = walk.NewAction()
	g.tray.aboutAction.SetText("About SnixConnect...")
	tryActions = append(tryActions, g.tray.aboutAction)
	tryActions = append(tryActions, walk.NewSeparatorAction())

	g.tray.hideWinAction = walk.NewAction()
	g.tray.hideWinAction.SetText("Hide")
	g.tray.showWinAction = walk.NewAction()
	g.tray.showWinAction.SetText("Show")
	tryActions = append(tryActions, g.tray.hideWinAction, g.tray.showWinAction)
	tryActions = append(tryActions, walk.NewSeparatorAction())

	g.tray.connectAction = walk.NewAction()
	tryActions = append(tryActions, g.tray.connectAction)

	g.tray.statusAction = walk.NewAction()
	g.tray.statusAction.SetText("Status: Disconnected")
	g.tray.statusAction.SetEnabled(false)
	tryActions = append(tryActions, g.tray.statusAction)

	for i := len(tryActions) - 1; i >= 0; i-- {
		g.tray.trayIcon.ContextMenu().Actions().Add(tryActions[i])
	}

	g.tray.hideWinAction.Triggered().Attach(func() { g.hideWindow() })
	g.tray.showWinAction.Triggered().Attach(func() { g.showWindow() })
	g.tray.showWinAction.SetVisible(false)

	return
}

func (g *appTrayNotify) showHideAction() {
	g.showWinAction.SetVisible(false)
	g.hideWinAction.SetVisible(true)
}

func (g *appTrayNotify) showShowAction() {
	g.hideWinAction.SetVisible(false)
	g.showWinAction.SetVisible(true)
}

func (g *appTrayNotify) attachAbotAction(h walk.EventHandler) {
	t := g.aboutAction.Triggered()
	if t == nil {
		return
	}
	t.Attach(h)
}

func (g *appTrayNotify) attachConnectAction(h walk.EventHandler) {
	t := g.connectAction.Triggered()
	if t == nil {
		return
	}
	t.Attach(h)
}

func (g *appTrayNotify) attachExitAction(h walk.EventHandler) {
	t := g.exitAction.Triggered()
	if t == nil {
		return
	}
	t.Once(h)
}
