package gui

import (
	"fmt"
	"runtime"
	"snixconnect/internal/version"
	"strings"
	"sync"

	"snixconnect/pkg/walk"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const snixConnectVersion = version.SnixConnectVersion
const licenseSnixConnect = `Maintained by <a id="link" href="https://github.com/sina-ghaderi">Sina Ghaderi</a>
Copyright © 2023-2024 SNIX LLC. All Rights Reserved.`
const versionFormatText = "Client Version: %s\nDriver Version: %s\nGolang Version: %s"

type winAboutProperty struct {
	aboutDialog *walk.Dialog
	aboutIsOpen bool
	mutex       sync.Mutex
}

func (g *winAboutProperty) newAboutDialog() {
	if g.aboutIsOpen {
		winAdjustPosCenter(g.aboutDialog)
		win.SetForegroundWindow(g.aboutDialog.Handle())
		return
	}
	go func() {
		err := g.showAboutDialog()
		if err == nil {
			return
		}

		err = fmt.Errorf("error running showAboutDialog: %v", err)
		go winErrorBox(nil, err)
	}()
}

func (g *winAboutProperty) showAboutDialog() (err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	runtime.LockOSThread()
	g.aboutDialog, err = walk.NewDialogWithFixedSize(nil)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			g.aboutDialog.Dispose()
		}
	}()

	hbox := walk.NewHBoxLayout()
	hbox.SetMargins(walk.Margins{HNear: 20, VNear: 20, HFar: 25, VFar: 20})
	hbox.SetSpacing(5)
	g.aboutDialog.SetTitle("About SnixConnect")
	g.aboutDialog.SetLayout(hbox)

	imageView, err := walk.NewImageView(g.aboutDialog)
	if err != nil {
		return
	}

	dpiwin := g.aboutDialog.DPI()
	setIconForWidget(imageView, appMainIconName, dpiwin, iconSize128x128)
	setIconForWidget(g.aboutDialog, appAboutIconName, dpiwin, iconSize32x32)
	iconComposite, err := walk.NewComposite(g.aboutDialog)
	if err != nil {
		return
	}

	vbox := walk.NewVBoxLayout()
	vbox.SetSpacing(10)

	iconComposite.SetLayout(vbox)
	mainTextLable, err := walk.NewTextLabel(iconComposite)
	if err != nil {
		return
	}
	setFontForWidget(mainTextLable, "Sogo UI", 15, walk.FontBold)
	mainTextLable.SetText("SnixConnect VPN Client")
	addinfo, err := walk.NewTextLabel(iconComposite)
	if err != nil {
		return
	}
	setFontForWidget(addinfo, appFontFamily, 8, walk.FontBold)

	addinfo.SetText(formatVersions())
	addinfo.SetFocus()
	copyright, err := walk.NewLinkLabel(iconComposite)
	if err != nil {
		return
	}

	copyright.SetText(licenseSnixConnect)
	onClickFunc := func(link *walk.LinkLabelLink) {
		addinfo.SetFocus()
		win.ShellExecute(copyright.Handle(), nil,
			windows.StringToUTF16Ptr(link.URL()), nil, nil, win.SW_SHOWNORMAL)
	}

	copyright.LinkActivated().Attach(onClickFunc)
	g.aboutDialog.Synchronize(func() {
		g.aboutIsOpen = true
		winAdjustPosCenter(g.aboutDialog)
	})
	g.aboutDialog.Disposing().Attach(func() { g.aboutIsOpen = false })
	g.aboutDialog.Run()
	return
}

func osRuningInfo() string {
	win32sysInfo := windows.RtlGetVersion()
	var winSysType string
	switch win32sysInfo.ProductType {
	case 3:
		winSysType = " Server"
	case 2:
		winSysType = " Controller"
	}

	return fmt.Sprintf("Windows%s %d.%d.%d", winSysType,
		win32sysInfo.MajorVersion,
		win32sysInfo.MinorVersion, win32sysInfo.BuildNumber)
}

func formatVersions() string {
	runVersionn := strings.TrimPrefix(runtime.Version(), "go")
	DriverVersion := "driver"
	return fmt.Sprintf(versionFormatText,
		snixConnectVersion, DriverVersion,
		runVersionn)
}
