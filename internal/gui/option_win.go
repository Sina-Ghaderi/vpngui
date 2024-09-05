package gui

import (
	"fmt"
	"runtime"
	"snixconnect/pkg/walk"
	"sync"

	"github.com/lxn/win"
)

type winOptionProperty struct {
	settingDialog *walk.Dialog
	currentConfig *UserAppConfig
	settingIsOpen bool
	mutex         sync.Mutex
}

func (g *winOptionProperty) newSettingDialog() {
	if g.settingIsOpen {
		winAdjustPosCenter(g.settingDialog)
		win.SetForegroundWindow(g.settingDialog.Handle())
		return
	}

	go func() {
		err := g.showSettingDialog()
		if err == nil {
			return
		}

		err = fmt.Errorf("error running showAboutDialog: %v", err)
		go winErrorBox(nil, err)
	}()
}

func (g *winOptionProperty) showSettingDialog() (err error) {

	g.mutex.Lock()
	defer g.mutex.Unlock()
	runtime.LockOSThread()

	g.settingDialog, err = walk.NewDialogWithFixedSize(nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			g.settingDialog.Dispose()
		}
	}()

	d := g.settingDialog.DPI()
	setIconForWidget(g.settingDialog, appSettingIconName, d, iconSize32x32)
	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 9, VNear: 9, VFar: 9, HFar: 9})
	g.settingDialog.SetTitle("SnixConnect Settings")
	g.settingDialog.SetLayout(vbox)
	setFontForWidget(g.settingDialog, appFontFamily, 9, 0)

	groupBox, err := walk.NewGroupBox(g.settingDialog)
	if err != nil {
		return err
	}
	vboxgr := walk.NewVBoxLayout()
	vboxgr.SetMargins(walk.Margins{HNear: 10, VNear: 20, VFar: 20, HFar: 120})
	vboxgr.SetAlignment(walk.AlignHNearVNear)
	vboxgr.SetSpacing(0)
	groupBox.SetLayout(vboxgr)
	groupBox.SetTitle("Change SnixConnect Settings")

	credentials, err := walk.NewCheckBox(groupBox)
	if err != nil {
		return err
	}
	tlsSkipVerify, err := walk.NewCheckBox(groupBox)
	if err != nil {
		return err
	}
	credentials.SetText("Cache Credentials")
	credentials.SetToolTipText("Save credential to use in future connection attempts")
	tlsSkipVerify.SetText("Allow Insecure TLS Connection")
	tlsSkipVerify.SetToolTipText("Don't validate the server's certificate")
	buttonComposite, err := walk.NewComposite(g.settingDialog)
	if err != nil {
		return err
	}
	buttonLayout := walk.NewHBoxLayout()
	buttonLayout.SetMargins(walk.Margins{})
	buttonComposite.SetLayout(buttonLayout)
	if _, err := walk.NewHSpacer(buttonComposite); err != nil {
		return err
	}
	buttonOK, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return err
	}
	buttonCancel, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return err
	}

	buttonSaveHandler := func() {
		newconf := new(UserAppConfig)
		newconf.CredentialCache = credentials.Checked()
		newconf.SkipTLSVerify = tlsSkipVerify.Checked()
		if !newconf.CredentialCache {
			if err := removeUserCerdential(); err != nil {
				logger.Print(err)
				winErrorBox(g.settingDialog, err)
				g.settingDialog.Cancel()
			}
		}
		if err := saveUserAppConfig(newconf); err != nil {
			logger.Print(err)
			winErrorBox(g.settingDialog, err)
			g.settingDialog.Cancel()
		}

		g.currentConfig = newconf
		g.settingDialog.Accept()
	}

	buttonOK.SetText("OK")
	buttonCancel.SetText("Cancel")
	buttonOK.Clicked().Attach(buttonSaveHandler)
	buttonCancel.Clicked().Attach(g.settingDialog.Cancel)
	onButtonPressEnter(buttonCancel.KeyUp(), g.settingDialog.Cancel)
	onButtonPressEnter(buttonOK.KeyUp(), buttonSaveHandler)
	credentials.SetChecked(g.currentConfig.CredentialCache)
	tlsSkipVerify.SetChecked(g.currentConfig.SkipTLSVerify)

	g.settingDialog.Synchronize(func() {
		g.settingIsOpen = true
		winAdjustPosCenter(g.settingDialog)
	})
	g.settingDialog.Disposing().Attach(func() { g.settingIsOpen = false })
	g.settingDialog.Run()
	return nil
}
