package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

const (
	msgQueue = wmApp + iota
	msgTray

	timerIdle    = 1
	timerAnimate = 2
)

var (
	window         windows.HWND
	queue          []func() error
	knob           *controller
	trayIcon       *tray
	taskbarCreated uint32
)

func post(action func() error) {
	queue = append(queue, action)
	postMessage(window, msgQueue, 0, 0)
}

func run(action func() error) {
	defer func() {
		if r := recover(); r != nil {
			logf("%v", r)
		}
	}()
	if err := action(); err != nil {
		logf("%v", err)
	}
}

func wndProc(hwnd windows.HWND, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch {
	case message == msgQueue:
		for len(queue) > 0 {
			action := queue[0]
			queue = queue[1:]
			run(action)
		}
	case message == msgTray && lParam == wmRButtonUp:
		trayIcon.showMenu()
	case message == wmTimer && wParam == timerIdle:
		knob.idle()
	case message == wmTimer && wParam == timerAnimate:
		knob.overlay.tick()
	case message == taskbarCreated && trayIcon != nil:
		trayIcon.add()
	default:
		return defWindowProc(hwnd, message, wParam, lParam)
	}
	return 0
}

func main() {
	runtime.LockOSThread()

	if _, err := windows.CreateMutex(nil, true, utf16z("KnobMixer-single-instance")); err != nil {
		return
	}

	if err := start(); err != nil {
		logf("%v", err)
		return
	}

	var m msg
	for getMessage(&m, 0, 0, 0) > 0 {
		dispatchMessage(&m)
	}
	trayIcon.remove()
}

func start() error {
	setProcessDPIAware()
	scale = float64(getDpiForSystem()) / 96

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	var token uintptr
	if status := gdiplusStartup(&token, &gdiplusStartupInput{version: 1}, nil); status != 0 {
		return fmt.Errorf("could not start gdiplus: status %d", status)
	}

	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return err
	}
	class := wndClassEx{wndProc: windows.NewCallback(wndProc), instance: instance, className: utf16z("KnobMixer")}
	class.size = uint32(unsafe.Sizeof(class))
	if _, err := registerClassEx(&class); err != nil {
		return err
	}
	var err error
	window, err = createWindowEx(0, class.className, nil, 0, 0, 0, 0, 0, 0, 0, instance, nil)
	if err != nil {
		return err
	}

	overlay, err := newOverlay(class.className, instance)
	if err != nil {
		return err
	}
	knob = &controller{overlay: overlay, eaten: map[uint32]bool{}}
	if _, err := setWindowsHookEx(whKeyboardLL, windows.NewCallback(knob.hook), instance, 0); err != nil {
		return fmt.Errorf("could not install keyboard hook: %w", err)
	}

	taskbarCreated = registerWindowMessage(utf16z("TaskbarCreated"))
	trayIcon = newTray()
	return nil
}
