package gui

import (
	"fmt"
	"strings"

	"github.com/lxn/win"

	"snixconnect/pkg/walk"
	"snixconnect/pkg/walk/declarative"
)

const (
	mainWinWidth  = 468
	mainWinHeight = 200
	mainWinName   = "SnixConnect VPN Client"
)

type winMainProperty struct {
	tray           *appTrayNotify
	mainWindow     *walk.MainWindow
	serverLineEdit *walk.LineEdit
	settingsButton *walk.Action
	aboutButton    *walk.Action
	toolbarHandler *walk.ToolBar
	connProgress   *walk.ProgressBar
	viewLogButton  *walk.PushButton
	connStatusBox  *walk.Composite
	connectButton  *walk.PushButton
	connStatusMsg  *walk.Label
	connRxTxLable  [3]*walk.Label
}

func (g *winMainProperty) hideWindow() {
	if g.tray.windowsIsCore {
		handle := g.mainWindow.Handle()
		win.ShowWindow(handle, win.SW_MINIMIZE)
		return
	}

	g.mainWindow.Hide()
	g.tray.showShowAction()
}

func (g *winMainProperty) showWindow() {
	handle := g.mainWindow.Handle()
	defer win.SetForegroundWindow(handle)
	if g.tray.windowsIsCore {
		win.ShowWindow(handle, win.SW_RESTORE)
		return
	}

	if win.IsIconic(handle) {
		win.ShowWindow(handle, win.SW_RESTORE)
	} else {
		g.mainWindow.Show()
	}
	g.tray.showHideAction()

}

func (g *winMainProperty) setAppGuiTweaks() {
	handle := g.mainWindow.Handle()
	style := win.GetWindowLong(handle, win.GWL_STYLE)
	newStyle := style &^ (win.WS_MAXIMIZEBOX | win.WS_THICKFRAME)
	win.SetWindowLong(handle, win.GWL_STYLE, newStyle)
	win.DeleteMenu(win.GetSystemMenu(handle, false), win.SC_MAXIMIZE, win.MF_BYCOMMAND)
	win.ShowWindow(handle, win.SW_SHOW)
	winAdjustPosRightBottom(g.mainWindow)
	appDPI := int(win.GetDpiForWindow(handle))
	setIconForWidget(g.mainWindow, appMainIconName, 0, iconSize32x32)
	toolBarDpi := (appDPI / 7) + appDPI
	g.toolbarHandler.ApplyDPI(toolBarDpi)
	setIconForWidget(g.settingsButton, appSettingIconName, toolBarDpi, iconSize64x64)
	setIconForWidget(g.aboutButton, appAboutIconName, toolBarDpi, iconSize64x64)
	setFontForWidget(g.mainWindow, appFontFamily, 9, 0)
	setFontForWidget(g.serverLineEdit, appFontFamily, 10, 0)
	setFontForWidget(g.connStatusMsg, appFontFamily, 10, 0)
	g.connectButton.SetFocus()
}

func (g *winMainProperty) validateAddress() bool {
	addrText := strings.TrimSpace(g.serverLineEdit.Text())
	g.serverLineEdit.SetText(addrText)
	urlString, err := isvalidUrl(addrText)
	if err == nil {
		g.serverLineEdit.SetText(urlString)
		return true
	}

	logger.Print(err)
	err = fmt.Errorf("%v make sure you typed the address correctly", err)
	winErrorBox(g.mainWindow, err)
	g.showWindow()
	g.serverLineEdit.SetTextSelection(0, -1)
	g.serverLineEdit.SetFocus()
	return false
}

func (g *winMainProperty) newMainWindow() *declarative.MainWindow {
	return &declarative.MainWindow{
		AssignTo: &g.mainWindow,
		Title:    mainWinName,
		Size:     declarative.Size{Height: mainWinHeight, Width: mainWinWidth},
		MinSize:  declarative.Size{Height: mainWinHeight, Width: mainWinWidth},
		Layout:   declarative.VBox{Margins: declarative.Margins{Left: 9, Top: 9, Right: 9, Bottom: 4}},
		Children: []declarative.Widget{
			declarative.GroupBox{
				DoubleBuffering: true,
				Title:           "Connect To Server",
				Layout: declarative.Grid{
					Columns:   2,
					Alignment: declarative.AlignHNearVCenter,
					Margins:   declarative.Margins{Bottom: 4, Top: 4, Left: 7, Right: 7},
				},
				Children: []declarative.Widget{
					declarative.Composite{
						DoubleBuffering: true,
						Layout:          declarative.Grid{Columns: 2, Alignment: declarative.AlignHNearVCenter},
						Children: []declarative.Widget{
							declarative.LineEdit{
								DoubleBuffering: true,
								AssignTo:        &g.serverLineEdit,
								CueBanner:       "https://vpn-example.site[:443]/[usergroup]",
								MaxSize:         declarative.Size{Height: 23},
								MaxLength:       2048,
								OnTextChanged: func() {
									t := strings.TrimSpace(g.serverLineEdit.Text())
									if len(t) > 0 {
										g.connectButton.SetEnabled(true)
										g.tray.connectAction.SetEnabled(true)
										return
									}
									g.connectButton.SetEnabled(false)
									g.tray.connectAction.SetEnabled(false)
								},
							},

							declarative.PushButton{
								DoubleBuffering: true,
								AssignTo:        &g.connectButton,
							},
						},
					},
				},
			},

			declarative.GroupBox{
				Title:           "Connection Status",
				Layout:          declarative.VBox{},
				DoubleBuffering: true,
				Children: []declarative.Widget{
					declarative.Composite{
						DoubleBuffering: true,
						Layout: declarative.Grid{
							Columns:   3,
							Alignment: declarative.AlignHNearVCenter,
							Margins:   declarative.Margins{Bottom: 1, Top: 4, Left: 7, Right: 7},
						},
						Children: []declarative.Widget{
							declarative.Composite{
								Layout:             declarative.HBox{MarginsZero: true, SpacingZero: true},
								AlwaysConsumeSpace: true,
								Border:             true,
								Children: []declarative.Widget{
									declarative.HSpacer{},
									declarative.ProgressBar{
										AssignTo:        &g.connProgress,
										MarqueeMode:     true,
										MaxSize:         declarative.Size{Height: 23},
										Visible:         false,
										DoubleBuffering: true,
									},
									declarative.Composite{
										Layout: declarative.HBox{
											SpacingZero: true,
											Margins:     declarative.Margins{Top: 2, Bottom: 6},
										},
										AssignTo:        &g.connStatusBox,
										DoubleBuffering: true,
										Background:      declarative.SolidColorBrush{Color: 0xe6e4e5},
										Children: []declarative.Widget{
											declarative.HSpacer{},
											declarative.Label{
												DoubleBuffering: true,
												AssignTo:        &g.connStatusMsg,
												TextColor:       colorDisconnect,
												Text:            textDisconnected,
												Alignment:       declarative.AlignHCenterVCenter,
											},
											declarative.HSpacer{},
										},
									},
								},
							},

							declarative.PushButton{
								AssignTo:        &g.viewLogButton,
								DoubleBuffering: true,
								Text:            "View Logs",
							},
						},
					},

					declarative.Composite{
						DoubleBuffering: true,
						Layout:          declarative.VBox{Margins: declarative.Margins{Left: 7, Right: 7}, MarginsZero: true},
						Children: []declarative.Widget{
							declarative.Composite{
								DoubleBuffering: true,
								Layout: declarative.HBox{
									Margins:     declarative.Margins{},
									MarginsZero: true,
									Spacing:     3,
								},
								Children: []declarative.Widget{
									declarative.Label{
										DoubleBuffering: true,
										Text:            "Receive:",
									},

									declarative.Label{
										DoubleBuffering: true,
										AssignTo:        &g.connRxTxLable[0],
										Enabled:         false,
									},

									declarative.HSpacer{Size: 7},
									declarative.Label{
										DoubleBuffering: true,
										Text:            "Transmit:",
									},

									declarative.Label{
										DoubleBuffering: true,
										AssignTo:        &g.connRxTxLable[1],
										Enabled:         false,
									},

									declarative.HSpacer{Size: 7},

									declarative.Label{
										DoubleBuffering: true,
										Text:            "Uptime:",
									},

									declarative.Label{
										DoubleBuffering: true,
										AssignTo:        &g.connRxTxLable[2],
										Enabled:         false,
									},
									declarative.HSpacer{},
								},
							},
						},
					},
				},
			},

			declarative.Composite{
				DoubleBuffering: true,
				Layout:          declarative.HBox{MarginsZero: true, SpacingZero: true},
				Alignment:       declarative.AlignHNearVNear,
				Children: []declarative.Widget{
					declarative.ToolBar{
						DoubleBuffering: true,
						AssignTo:        &g.toolbarHandler,
						Orientation:     declarative.Vertical,
						ButtonStyle:     declarative.ToolBarButtonImageOnly,
						Items: []declarative.MenuItem{
							declarative.Action{
								AssignTo: &g.settingsButton,
								Text:     "Setting and Optaions",
							},

							declarative.Action{
								AssignTo: &g.aboutButton,
								Text:     "About SnixConnect",
							},
						},
					},
				},
			},
		},
	}

}
