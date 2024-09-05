package gui

import (
	"fmt"
	"runtime"
	"snixconnect/pkg/walk"
	"sync"

	"github.com/lxn/win"
)

type winBannerProperty struct {
	bannerDialog *walk.Dialog
	bannerIsOpen bool
	mutex        sync.Mutex
}

func (g *winBannerProperty) newBannerDialog(banner string) {
	g.closeDialog(walk.DlgCmdAbort)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	go func() {
		err := g.showBannerDialog(banner)
		if err == nil {
			return
		}
		err = fmt.Errorf("error running showBannerDialog: %v", err)
		go winErrorBox(nil, err)
	}()
}

func (g *winBannerProperty) showBannerDialog(banner string) (err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	runtime.LockOSThread()

	g.bannerDialog, err = walk.NewDialogWithStyle(nil,
		win.WS_POPUPWINDOW, win.WS_EX_TOPMOST)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			g.bannerDialog.Dispose()
		}
	}()
	setIconForWidget(g.bannerDialog, appAboutIconName,
		g.bannerDialog.DPI(), iconSize32x32)
	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 9, VNear: 9, VFar: 9, HFar: 9})
	g.bannerDialog.SetTitle("SnixConnect Banner")
	g.bannerDialog.SetLayout(vbox)
	setFontForWidget(g.bannerDialog, appFontFamily, 9, 0)

	groupBox, err := walk.NewGroupBox(g.bannerDialog)
	if err != nil {
		return
	}
	vboxgr := walk.NewVBoxLayout()
	vboxgr.SetMargins(walk.Margins{HNear: 10, VNear: 20, VFar: 20, HFar: 10})
	vboxgr.SetAlignment(walk.AlignHNearVNear)
	groupBox.SetLayout(vboxgr)
	groupBox.SetTitle("Message From Server")

	bannerText, err := walk.NewTextEdit(groupBox)
	if err != nil {
		return
	}

	setFontForWidget(bannerText, "Segoe UI", 9, 0)

	bannerText.SetMinMaxSize(walk.Size{Width: 360, Height: 190}, walk.Size{})
	bannerText.SetTextAlignment(walk.AlignCenter)
	bannerText.SetReadOnly(true)
	bannerText.SetText(banner + "\n")
	bannerText.SetMaxLength(1024)

	buttonComposite, err := walk.NewComposite(g.bannerDialog)
	if err != nil {
		return
	}
	buttonLayout := walk.NewHBoxLayout()
	buttonLayout.SetMargins(walk.Margins{})
	buttonComposite.SetLayout(buttonLayout)
	if _, err = walk.NewHSpacer(buttonComposite); err != nil {
		return
	}

	buttonOK, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return
	}

	bannerText.SetFocus()
	buttonOK.SetText("OK")
	buttonOK.Clicked().Attach(g.bannerDialog.Accept)
	onButtonPressEnter(buttonOK.KeyUp(), g.bannerDialog.Accept)
	g.bannerDialog.Synchronize(func() {
		g.bannerIsOpen = true
		winAdjustPos(g.bannerDialog, 1.8)
	})

	g.bannerDialog.Disposing().Attach(func() { g.bannerIsOpen = false })
	if g.bannerDialog.Run() == walk.DlgCmdAbort {
		logger.Print("banner dialog message terminated by another thread")
	}
	return
}

func (g *winBannerProperty) closeDialog(c int) {
	if !g.bannerIsOpen {
		return
	}

	g.bannerDialog.SetResult(c)
	win.PostMessage(g.bannerDialog.Handle(), win.WM_CLOSE, 0, 0)
}
