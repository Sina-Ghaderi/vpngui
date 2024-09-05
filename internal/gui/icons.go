package gui

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"math"
	"sync"

	"snixconnect/pkg/walk"

	"golang.org/x/image/draw"
)

const (
	appMainIconName       = "icon/snixconnect.png"
	appSettingIconName    = "icon/setting.png"
	appAboutIconName      = "icon/about.png"
	appViewLogIconName    = "icon/logs.png"
	appCredentialIconName = "icon/credential.png"

	connIconConnected    = "icon/tray_connect.png"
	connIconDisconnected = "icon/tray_disconnect.png"
	connIconFailed       = "icon/tray_error.png"
	connIconReconnect    = "icon/tray_reconnect.png"
	appIconGreenName     = "icon/green_snix.png"
	appIconOrangeName    = "icon/orange_snix.png"
)

const (
	iconSize32x32   = 32
	iconSize64x64   = 64
	iconSize128x128 = 128
)

//go:embed icon/*
var iconFolder embed.FS

type iconPathDpi struct {
	path string
	dpi  int
	size int
}

var iconMapImage = struct {
	iconMap map[iconPathDpi]*walk.Icon
	sync.Mutex
}{iconMap: make(map[iconPathDpi]*walk.Icon)}

func loadIconByName(name string) image.Image {
	imgByte, err := iconFolder.ReadFile(name)
	if err != nil {
		return nil
	}

	img, err := png.Decode(bytes.NewReader(imgByte))
	if err != nil {
		return nil
	}
	return img
}

func loadTrayStatusIcon(status StatusFlag, dpi int) walk.Image {
	iconName := appMainIconName
	switch status {
	case FlagConnected:
		iconName = appIconGreenName
	case FlagReconnecting, FlagConnecting:
		iconName = appIconOrangeName
	}
	statIcon := loadWalkIconByname(iconName, dpi, iconSize32x32)
	if statIcon == nil {
		return nil
	}
	return statIcon
}

func loadWalkIconByname(name string, dpi, size int) *walk.Icon {
	iconMapImage.Lock()
	defer iconMapImage.Unlock()
	pathDpi := iconPathDpi{path: name, dpi: dpi, size: size}

	icon, ok := iconMapImage.iconMap[pathDpi]
	if ok {
		return icon
	}

	img := loadIconByName(name)
	if img == nil {
		return nil
	}
	if size > img.Bounds().Max.X || size > img.Bounds().Max.Y {
		m := math.Min(float64(img.Bounds().Max.X), float64(img.Bounds().Max.Y))
		size = int(m)
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)

	var err error
	icon, err = walk.NewIconFromImageForDPI(dst, dpi)
	if err != nil {
		return nil
	}
	iconMapImage.iconMap[pathDpi] = icon
	return icon
}

func setIconForWidget(window any, path string, dpi, size int) {
	icon := loadWalkIconByname(path, dpi, size)
	if icon == nil {
		return
	}
	iconSetter, ok := window.(interface{ SetIcon(walk.Image) error })
	if ok {
		iconSetter.SetIcon(icon)
		return
	}

	imageSetter, ok := window.(interface{ SetImage(walk.Image) error })
	if ok {
		imageSetter.SetImage(icon)
		return
	}

}

func setFontForWidget(window interface{ SetFont(*walk.Font) },
	fontname string, size int, style walk.FontStyle) {
	textfont, err := walk.NewFont(fontname, size, walk.FontStyle(style))
	if err != nil {
		return
	}
	window.SetFont(textfont)
}
