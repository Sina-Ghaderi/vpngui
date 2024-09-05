package gui

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"snixconnect/pkg/walk"

	"github.com/lxn/win"
)

type winLogsProperty struct {
	updateDetails  atomic.Value
	updateLogTable atomic.Value
	logModel       *appLogViewModel
	logDialog      *walk.Dialog
	connStats      *ConnectionStats
	logIsOpen      bool
	mutex          sync.Mutex
}

func (g *winLogsProperty) loggerFunc(s string) { g.updateLogTable.Load().(func(string))(s) }
func (g *winLogsProperty) detailsUpdater()     { g.updateDetails.Load().(func())() }

type ConnectionStats struct {
	ConnectedSince   time.Time
	Gateway          string
	MTU              uint16
	DNS              []string
	TunIPv4, Netmask string
	RX, TX           func() uint64
}

func layer1BoxLayout() walk.Layout {
	layout := walk.NewVBoxLayout()
	layout.SetAlignment(walk.AlignHCenterVCenter)
	layout.SetMargins(walk.Margins{HNear: 15, HFar: 15, VNear: 15, VFar: 15})
	layout.SetSpacing(0)
	return layout
}

func layer2BoxLayout() walk.Layout {
	layout := walk.NewHBoxLayout()
	layout.SetAlignment(walk.AlignHFarVFar)
	layout.SetMargins(walk.Margins{})
	layout.SetSpacing(0)
	return layout
}

func (g *winLogsProperty) newViewLogsDialog() {
	if !g.logIsOpen {
		go g.runShowLogsDialog()
		return
	}

	winAdjustPosCenter(g.logDialog)
	win.SetForegroundWindow(g.logDialog.Handle())
}

func (g *winLogsProperty) runShowLogsDialog() {
	err := g.showLogsDialog()
	if err == nil {
		return
	}

	err = fmt.Errorf("error running showLogsDialog: %v", err)
	go winErrorBox(nil, err)
}

func (g *winLogsProperty) showLogsDialog() (err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	runtime.LockOSThread()
	g.logDialog, err = walk.NewDialogWithStyle(nil, win.WS_POPUPWINDOW, 0)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			g.logDialog.Dispose()
		}
	}()

	setIconForWidget(g.logDialog, appViewLogIconName, g.logDialog.DPI(), iconSize32x32)
	setFontForWidget(g.logDialog, appFontFamily, 9, 0)
	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 9, VNear: 9, VFar: 9, HFar: 9})
	g.logDialog.SetTitle("SnixConnect Log And Details")
	g.logDialog.SetLayout(vbox)

	groupBox, err := walk.NewGroupBox(g.logDialog)
	if err != nil {
		return
	}
	groupBox.SetDoubleBuffering(true)
	vboxgr := walk.NewHBoxLayout()
	vboxgr.SetMargins(walk.Margins{HNear: 15, VNear: 2, VFar: 2, HFar: 15})
	vboxgr.SetAlignment(walk.AlignHCenterVCenter)
	vboxgr.SetSpacing(2)
	groupBox.SetLayout(vboxgr)
	groupBox.SetTitle("Connection Details")

	comp1, err := walk.NewComposite(groupBox)
	if err != nil {
		return
	}
	spacer, err := walk.NewHSpacer(groupBox)
	if err != nil {
		return
	}
	spacer.SetMinMaxSize(walk.Size{Width: 200}, walk.Size{})
	comp2, err := walk.NewComposite(groupBox)
	if err != nil {
		return
	}
	comp1.SetLayout(layer1BoxLayout())
	comp2.SetLayout(layer1BoxLayout())
	comp1.SetDoubleBuffering(true)
	comp2.SetDoubleBuffering(true)

	copm11, err := walk.NewComposite(comp1)
	if err != nil {
		return
	}
	copm12, err := walk.NewComposite(comp1)
	if err != nil {
		return
	}
	copm13, err := walk.NewComposite(comp1)
	if err != nil {
		return
	}
	copm14, err := walk.NewComposite(comp1)
	if err != nil {
		return
	}
	copm21, err := walk.NewComposite(comp2)
	if err != nil {
		return
	}
	copm22, err := walk.NewComposite(comp2)
	if err != nil {
		return
	}
	copm23, err := walk.NewComposite(comp2)
	if err != nil {
		return
	}
	copm24, err := walk.NewComposite(comp2)
	if err != nil {
		return
	}
	copm11.SetLayout(layer2BoxLayout())
	copm12.SetLayout(layer2BoxLayout())
	copm13.SetLayout(layer2BoxLayout())
	copm14.SetLayout(layer2BoxLayout())
	copm21.SetLayout(layer2BoxLayout())
	copm22.SetLayout(layer2BoxLayout())
	copm24.SetLayout(layer2BoxLayout())
	copm23.SetLayout(layer2BoxLayout())
	copm11.SetDoubleBuffering(true)
	copm12.SetDoubleBuffering(true)
	copm13.SetDoubleBuffering(true)
	copm14.SetDoubleBuffering(true)
	copm21.SetDoubleBuffering(true)
	copm22.SetDoubleBuffering(true)
	copm24.SetDoubleBuffering(true)
	copm23.SetDoubleBuffering(true)

	lbIPv4, err := textLableValue(copm11, "IPv4 Address:")
	if err != nil {
		return
	}

	lbNetmask, err := textLableValue(copm12, "Netmask:")
	if err != nil {
		return
	}

	lbGateway, err := textLableValue(copm21, "Link Gateway:")
	if err != nil {
		return
	}

	lbLinkMTU, err := textLableValue(copm22, "Link MTU:")
	if err != nil {
		return
	}

	var lbDNS [4]*walk.TextLabel

	lbDNS[0], err = textLableValue(copm13, "Nameserver #1:")
	if err != nil {
		return
	}

	lbDNS[1], err = textLableValue(copm14, "Nameserver #2:")
	if err != nil {
		return
	}

	lbDNS[2], err = textLableValue(copm23, "Nameserver #3:")
	if err != nil {
		return
	}

	lbDNS[3], err = textLableValue(copm24, "Nameserver #4:")
	if err != nil {
		return
	}

	connStatHandler := func() {
		g.logDialog.SetSuspended(true)
		defer g.logDialog.SetSuspended(false)

		fillConnectionStats(lbIPv4, g.connStats.TunIPv4)
		fillConnectionStats(lbGateway, g.connStats.Gateway)
		fillConnectionStats(lbNetmask, g.connStats.Netmask)
		switch {
		case g.connStats.MTU > 0:
			lbLinkMTU.SetText(fmt.Sprint(g.connStats.MTU))
			lbLinkMTU.SetEnabled(true)
		default:
			lbLinkMTU.SetEnabled(false)
			lbLinkMTU.SetText("Not Available")
		}

		for i, dnsLable := range lbDNS {
			if len(g.connStats.DNS) > i {
				fillConnectionStats(dnsLable, g.connStats.DNS[i])
				continue
			}
			dnsLable.SetEnabled(false)
			dnsLable.SetText("Not Available")
		}
	}

	g.updateDetails.Store(connStatHandler)
	logTable, err := newLoggingTable(g.logDialog)
	if err != nil {
		return
	}

	logTable.SetModel(g.logModel)
	if len(g.logModel.items) > 0 {
		logTable.EnsureItemVisible(len(g.logModel.items) - 1)
	}

	g.updateLogTable.Store(func(logLine string) {
		tableAtbot := (len(g.logModel.items) == 0 ||
			logTable.ItemVisible(len(g.logModel.items)-1)) &&
			len(logTable.SelectedIndexes()) <= 1
		updateRows := g.logModel.addLogToItems(logLine)
		g.logModel.PublishRowsReset()
		if updateRows {
			ln := len(g.logModel.items) - 1
			g.logModel.PublishRowsChanged(0, ln)
		}
		if tableAtbot {
			ln := len(g.logModel.items)
			logTable.EnsureItemVisible(ln - 1)
		}
	})

	buttonComposite, err := walk.NewComposite(g.logDialog)
	if err != nil {
		return
	}
	buttonComposite.SetDoubleBuffering(true)
	buttonLayout := walk.NewHBoxLayout()
	buttonLayout.SetMargins(walk.Margins{})
	buttonComposite.SetLayout(buttonLayout)
	if _, err = walk.NewHSpacer(buttonComposite); err != nil {
		return
	}
	buttonSave, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return
	}

	buttonClose, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return
	}

	buttonSave.SetDoubleBuffering(true)
	buttonClose.SetDoubleBuffering(true)
	buttonSave.SetText("Export")
	buttonClose.SetText("Close")

	saveToFileHandler := func() {
		defer func() { g.logDialog.SetFocus() }()
		fileSelect := new(walk.FileDialog)
		fileSelect.Title = "Export Log To File"
		d := time.Now().Format("2006-01-02T150405")
		fileSelect.FilePath = fmt.Sprintf("snixconnect-log-%s.txt", d)
		fileSelect.Filter = "Text Files (*.txt)|*.txt|All Files (*.*)|*.*"
		accepted, err := fileSelect.ShowSave(g.logDialog)
		if err != nil {
			logger.Print(err)
			return
		}
		if !accepted {
			return
		}
		if fileSelect.FilterIndex == 1 &&
			!strings.HasSuffix(fileSelect.FilePath, ".txt") {
			fileSelect.FilePath = fileSelect.FilePath + ".txt"
		}
		err = g.logModel.saveLogsToFile(fileSelect.FilePath)
		if err != nil {
			logger.Print(err)
			winErrorBox(g.logDialog, err)
			return
		}
	}

	buttonClose.Clicked().Attach(g.logDialog.Cancel)
	buttonSave.Clicked().Attach(saveToFileHandler)
	onButtonPressEnter(buttonSave.KeyUp(), saveToFileHandler)
	onButtonPressEnter(buttonClose.KeyUp(), g.logDialog.Cancel)

	closingFunc := func() {
		g.updateDetails.Store(func() {})
		flog := func(l string) { g.logModel.addLogToItems(l) }
		g.updateLogTable.Store(flog)
		g.logIsOpen = false
	}

	syncFunc := func() {
		g.logIsOpen = true
		winAdjustPosCenter(g.logDialog)
	}

	g.detailsUpdater()
	g.logDialog.Disposing().Attach(closingFunc)
	g.logDialog.Synchronize(syncFunc)
	buttonClose.SetFocus()
	g.logDialog.Run()
	return nil
}

func textLableValue(p walk.Container, l string) (*walk.TextLabel, error) {
	lable, err := walk.NewLabel(p)
	if err != nil {
		return nil, err
	}
	lable.SetDoubleBuffering(true)
	lable.SetText(l)
	space, err := walk.NewHSpacer(p)
	if err != nil {
		return nil, err
	}
	space.SetMinMaxSize(walk.Size{Width: 35}, walk.Size{})
	valuelb, err := walk.NewTextLabel(p)
	if err != nil {
		return nil, err
	}
	valuelb.SetDoubleBuffering(true)
	return valuelb, err
}

func fillConnectionStats(tl *walk.TextLabel, s string) {
	if len(s) > 0 {
		tl.SetEnabled(true)
		tl.SetText(s)
		return
	}
	tl.SetEnabled(false)
	tl.SetText("Not Available")
}

func newLoggingTable(p walk.Form) (*walk.TableView, error) {
	const (
		tableWidth  = 865
		tableHeight = 325
	)

	t_size := walk.Size{Width: tableWidth, Height: tableHeight}

	logTable, err := walk.NewTableView(p)
	if err != nil {
		return nil, err
	}

	columTime := walk.NewTableViewColumn()
	columTime.SetTitle("Time")
	columTime.SetDataMember("Stamp")
	columTime.SetFormat("15:04:05.000")
	columTime.SetWidth(96)

	columLine := walk.NewTableViewColumn()
	columLine.SetTitle("Log message")
	columLine.SetDataMember("Line")

	logTable.Columns().Add(columTime)
	logTable.Columns().Add(columLine)

	logTable.SetAlternatingRowBG(true)
	logTable.SetLastColumnStretched(true)
	logTable.SetGridlines(true)
	logTable.SetMinMaxSize(t_size, t_size)

	return logTable, nil

}
