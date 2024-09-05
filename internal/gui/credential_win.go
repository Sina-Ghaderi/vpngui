package gui

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"snixconnect/pkg/walk"

	"github.com/lxn/win"
)

const (
	usernameMaxLen = 250
	passwordMaxLen
)

type winCredProperty struct {
	c                *userCredential
	credentialDialog *walk.Dialog
	credIsOpen       bool
	mutex            sync.Mutex
}

type GroupSelect struct {
	Name         string
	FriendlyName string
}

type groupSelectModel struct {
	walk.ListModelBase
	items []GroupSelect
}

func (m *groupSelectModel) ItemCount() int { return len(m.items) }
func (m *groupSelectModel) Value(index int) interface{} {
	return m.items[index].FriendlyName
}

func (g *winCredProperty) newCredentialDialog(rch chan int, glist []GroupSelect, banner string) {
	g.closeDialog(walk.DlgCmdAbort)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	go func() {
		err := g.showCredentialDialog(rch, glist, banner)
		if err == nil {
			return
		}
		err = fmt.Errorf("error running showCredentialDialog: %v", err)
		go winErrorBox(nil, err)
	}()
}

func (g *winCredProperty) showCredentialDialog(rch chan int, list []GroupSelect, banner string) (err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	runtime.LockOSThread()
	defer close(rch)

	g.credentialDialog, err = walk.NewDialogWithStyle(nil,
		win.WS_POPUPWINDOW, win.WS_EX_TOPMOST)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			g.credentialDialog.Dispose()
		}
	}()

	setFontForWidget(g.credentialDialog, appFontFamily, 9, 0)
	setIconForWidget(g.credentialDialog, appCredentialIconName,
		g.credentialDialog.DPI(), iconSize32x32)

	if err := preLoginBanner(g.credentialDialog, banner); err != nil {
		return err
	}

	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 9, VNear: 9, VFar: 9, HFar: 9})
	g.credentialDialog.SetTitle("Credentials")
	g.credentialDialog.SetLayout(vbox)

	groupBox, err := walk.NewGroupBox(g.credentialDialog)
	if err != nil {
		return
	}
	vboxgr := walk.NewVBoxLayout()
	vboxgr.SetMargins(walk.Margins{HNear: 10, VNear: 20, VFar: 20, HFar: 10})
	vboxgr.SetAlignment(walk.AlignHNearVNear)
	vboxgr.SetSpacing(2)
	groupBox.SetLayout(vboxgr)
	groupBox.SetTitle("Connection Credentials")

	gSelect, err := g.setupGroupList(groupBox, list)
	if err != nil {
		return
	}

	lbUser, err := walk.NewLabel(groupBox)
	if err != nil {
		return
	}

	userLine, err := walk.NewLineEdit(groupBox)
	if err != nil {
		return
	}

	vSpace, err := walk.NewVSpacer(groupBox)
	if err != nil {
		return
	}

	lbPass, err := walk.NewLabel(groupBox)
	if err != nil {
		return
	}

	passLine, err := walk.NewLineEdit(groupBox)
	if err != nil {
		return
	}

	lbUser.SetText("Username:")
	lbPass.SetText("Password:")
	vSpace.SetMinMaxSize(walk.Size{Height: 5}, walk.Size{})
	userLine.SetText(g.c.Username)
	passLine.SetText(g.c.Password)
	passLine.SetPasswordMode(true)
	passLine.SetMaxLength(passwordMaxLen)
	userLine.SetMaxLength(usernameMaxLen)

	userLine.SetMinMaxSize(walk.Size{Width: 250}, walk.Size{})
	passLine.SetMinMaxSize(walk.Size{Width: 250}, walk.Size{})

	buttonComposite, err := walk.NewComposite(g.credentialDialog)
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
	buttonCancel, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return
	}

	okHandler := func() {
		g.c.Username = strings.TrimSpace(userLine.Text())
		g.c.Password = strings.TrimSpace(passLine.Text())
		g.c.Group = gSelect()
		g.credentialDialog.Accept()
	}

	buttonCancel.SetText("Cancel")
	buttonOK.SetText("OK")
	buttonOK.Clicked().Attach(okHandler)
	buttonCancel.Clicked().Attach(g.credentialDialog.Cancel)
	buttonOK.SetFocus()
	onButtonPressEnter(buttonCancel.KeyUp(), g.credentialDialog.Cancel)
	onButtonPressEnter(userLine.KeyUp(), func() { passLine.SetFocus() })
	onButtonPressEnter(passLine.KeyUp(), func() { buttonOK.SetFocus() })
	onButtonPressEnter(buttonOK.KeyUp(), okHandler)

	g.credentialDialog.Synchronize(func() {
		g.credIsOpen = true
		winAdjustPos(g.credentialDialog, 1.8)
	})

	g.credentialDialog.Disposing().Attach(func() { g.credIsOpen = false })
	rch <- g.credentialDialog.Run()
	return
}

func (g *winCredProperty) setupGroupList(p walk.Container, list []GroupSelect) (func() string, error) {
	model := &groupSelectModel{items: []GroupSelect{}}
	selectIndex := 0

	for _, v := range list {
		if len(v.Name) == 0 {
			continue
		}
		if len(v.FriendlyName) == 0 {
			v.FriendlyName = v.Name
		}
		model.items = append(model.items, v)
		if v.Name == g.c.Group {
			selectIndex = len(model.items) - 1
		}
	}

	if len(model.items) < 1 {
		return func() (s string) { return }, nil
	}

	lbGroup, err := walk.NewLabel(p)
	if err != nil {
		return nil, err
	}

	lbGroup.SetText("Group:")

	groups, err := walk.NewDropDownBox(p)
	if err != nil {
		return nil, err
	}

	spacer, err := walk.NewVSpacer(p)
	if err != nil {
		return nil, err
	}
	spacer.SetMinMaxSize(walk.Size{Height: 5}, walk.Size{})

	groups.SetModel(model)
	f := func() string {
		return model.items[groups.CurrentIndex()].Name
	}

	groups.SetCurrentIndex(selectIndex)
	return f, nil

}

func preLoginBanner(p walk.Container, banner string) error {

	if len(banner) == 0 {
		return nil
	}

	groupBox, err := walk.NewGroupBox(p)
	if err != nil {
		return err
	}
	vboxgr := walk.NewVBoxLayout()
	vboxgr.SetMargins(walk.Margins{HNear: 10, VNear: 20, VFar: 20, HFar: 10})
	vboxgr.SetAlignment(walk.AlignHNearVNear)
	vboxgr.SetSpacing(2)
	groupBox.SetLayout(vboxgr)
	groupBox.SetTitle("Message From Server")

	bannerText, err := walk.NewTextEdit(groupBox)
	if err != nil {
		return err
	}

	setFontForWidget(bannerText, "Segoe UI", 9, 0)
	bannerText.SetMinMaxSize(walk.Size{Width: 250, Height: 95}, walk.Size{})
	bannerText.SetTextAlignment(walk.AlignCenter)
	bannerText.SetReadOnly(true)
	bannerText.SetMaxLength(250)
	bannerText.SetText(banner + "\n")
	return nil
}

func (g *winCredProperty) closeDialog(c int) {
	if !g.credIsOpen {
		return
	}
	g.credentialDialog.SetResult(c)
	win.PostMessage(g.credentialDialog.Handle(), win.WM_CLOSE, 0, 0)
}
