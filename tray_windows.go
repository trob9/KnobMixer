package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runName = "KnobMixer"

	menuAutostart = 1
	menuQuit      = 2
)

type tray struct {
	data      notifyIconData
	menu      windows.Handle
	autostart bool
}

func newTray() *tray {
	t := &tray{menu: createPopupMenu(), autostart: isAutostart()}
	check := uint32(mfUnchecked)
	if t.autostart {
		check = mfChecked
	}
	appendMenu(t.menu, mfString|check, menuAutostart, utf16z("Start with Windows"))
	appendMenu(t.menu, mfString, menuQuit, utf16z("Quit"))

	t.data = notifyIconData{
		size:            uint32(unsafe.Sizeof(t.data)),
		hwnd:            window,
		flags:           nifMessage | nifIcon | nifTip,
		callbackMessage: msgTray,
		icon:            knobIcon(),
	}
	copy(t.data.tip[:], windows.StringToUTF16("KnobMixer - click your knob to open"))
	t.add()
	return t
}

func (t *tray) add() {
	shellNotifyIcon(nimAdd, &t.data)
}

func (t *tray) remove() {
	shellNotifyIcon(nimDelete, &t.data)
}

func (t *tray) showMenu() {
	var cursor point
	getCursorPos(&cursor)
	setForegroundWindow(window)
	command := trackPopupMenu(t.menu, tpmRightButton|tpmNoNotify|tpmReturnCmd, cursor.x, cursor.y, 0, window, nil)
	postMessage(window, wmNull, 0, 0)

	switch command {
	case menuAutostart:
		t.autostart = !t.autostart
		check := uint32(mfUnchecked)
		if t.autostart {
			check = mfChecked
		}
		checkMenuItem(t.menu, menuAutostart, check)
		setAutostart(t.autostart)
	case menuQuit:
		postQuitMessage(0)
	}
}

func isAutostart() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetValue(runName, nil)
	return err != registry.ErrNotExist
}

func setAutostart(on bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	if !on {
		key.DeleteValue(runName)
		return
	}
	path, err := os.Executable()
	if err != nil {
		return
	}
	key.SetStringValue(runName, `"`+path+`"`)
}

func knobIcon() windows.Handle {
	var bmp, graphics uintptr
	gdipCreateBitmapFromScan0(32, 32, 0, pixelFormat32bppARGB, nil, &bmp)
	gdipGetImageGraphicsContext(bmp, &graphics)
	gdipSetSmoothingMode(graphics, smoothingAntiAlias)

	fill := func(argb uint32, draw func(brush uintptr)) {
		var brush uintptr
		gdipCreateSolidFill(argb, &brush)
		draw(brush)
		gdipDeleteBrush(brush)
	}
	fill(0xFF5CC8FF, func(brush uintptr) { gdipFillEllipseI(graphics, brush, 2, 2, 28, 28) })
	fill(0xFF16161A, func(brush uintptr) { gdipFillEllipseI(graphics, brush, 6, 6, 20, 20) })
	fill(0xFFFFFFFF, func(brush uintptr) { gdipFillRectangleI(graphics, brush, 15, 5, 3, 10) })

	var icon windows.Handle
	gdipCreateHICONFromBitmap(bmp, &icon)
	gdipDeleteGraphics(graphics)
	gdipDisposeImage(bmp)
	return icon
}
