package main

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
	"golang.org/x/sys/windows"
)

type entry struct {
	name     string
	icon     image.Image
	endpoint *wca.IAudioEndpointVolume
	sessions []*wca.ISimpleAudioVolume
}

func (e *entry) volume() (float32, error) {
	var volume float32
	if e.endpoint != nil {
		err := e.endpoint.GetMasterVolumeLevelScalar(&volume)
		return volume, err
	}
	for _, s := range e.sessions {
		var v float32
		if err := s.GetMasterVolume(&v); err != nil {
			return 0, err
		}
		volume = max(volume, v)
	}
	return volume, nil
}

func (e *entry) muted() (bool, error) {
	if e.endpoint == nil {
		return false, nil
	}
	var mute uint32
	err := e.endpoint.GetMute((*bool)(unsafe.Pointer(&mute)))
	return mute != 0, err
}

func (e *entry) nudge(delta float32) error {
	volume, err := e.volume()
	if err != nil {
		return err
	}
	volume = min(max(volume+delta, 0), 1)
	if e.endpoint != nil {
		muted, err := e.muted()
		if err != nil {
			return err
		}
		if delta > 0 && muted {
			if err := e.endpoint.SetMute(false, ole.IID_NULL); err != nil {
				return err
			}
		}
		return e.endpoint.SetMasterVolumeLevelScalar(volume, ole.IID_NULL)
	}
	for _, s := range e.sessions {
		if err := s.SetMasterVolume(volume, ole.IID_NULL); err != nil {
			return err
		}
	}
	return nil
}

type audio struct {
	held []interface{ Release() int32 }
}

func (a *audio) release() {
	for _, h := range a.held {
		h.Release()
	}
	a.held = nil
}

func (a *audio) refresh() ([]*entry, error) {
	a.release()

	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &enumerator); err != nil {
		return nil, err
	}
	a.held = append(a.held, enumerator)

	var device *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EMultimedia, &device); err != nil {
		return nil, err
	}
	a.held = append(a.held, device)

	var endpoint *wca.IAudioEndpointVolume
	if err := device.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &endpoint); err != nil {
		return nil, err
	}
	a.held = append(a.held, endpoint)

	entries := []*entry{{name: "Master", endpoint: endpoint}}

	var devices *wca.IMMDeviceCollection
	if err := enumerator.EnumAudioEndpoints(wca.ERender, wca.DEVICE_STATE_ACTIVE, &devices); err != nil {
		return nil, err
	}
	a.held = append(a.held, devices)

	var deviceCount uint32
	if err := devices.GetCount(&deviceCount); err != nil {
		return nil, err
	}

	apps := map[string]*entry{}
	for i := range deviceCount {
		var device *wca.IMMDevice
		if err := devices.Item(i, &device); err != nil {
			return nil, err
		}
		a.held = append(a.held, device)

		var manager *wca.IAudioSessionManager2
		if err := device.Activate(wca.IID_IAudioSessionManager2, wca.CLSCTX_ALL, nil, &manager); err != nil {
			return nil, err
		}
		a.held = append(a.held, manager)

		var sessions *wca.IAudioSessionEnumerator
		if err := manager.GetSessionEnumerator(&sessions); err != nil {
			return nil, err
		}
		a.held = append(a.held, sessions)

		var sessionCount int
		if err := sessions.GetCount(&sessionCount); err != nil {
			return nil, err
		}

		for j := range sessionCount {
			var control *wca.IAudioSessionControl
			if err := sessions.GetSession(j, &control); err != nil {
				return nil, err
			}
			a.held = append(a.held, control)

			var state uint32
			if err := control.GetState(&state); err != nil {
				return nil, err
			}

			unknown, err := control.QueryInterface(wca.IID_IAudioSessionControl2)
			if err != nil {
				return nil, err
			}
			control2 := (*wca.IAudioSessionControl2)(unsafe.Pointer(unknown))
			a.held = append(a.held, control2)

			if state == wca.AudioSessionStateExpired || control2.IsSystemSoundsSession() == nil {
				continue
			}

			var pid uint32
			if err := failure(control2.GetProcessId(&pid)); err != nil {
				return nil, err
			}

			key := appKey(pid)
			app, ok := apps[key]
			if !ok {
				var displayName string
				if err := control.GetDisplayName(&displayName); err != nil {
					return nil, err
				}
				app = &entry{}
				app.name, app.icon = describe(pid, displayName)
				apps[key] = app
				entries = append(entries, app)
			}

			unknown, err = control.QueryInterface(wca.IID_ISimpleAudioVolume)
			if err != nil {
				return nil, err
			}
			volume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(unknown))
			a.held = append(a.held, volume)
			app.sessions = append(app.sessions, volume)
		}
	}
	return entries, nil
}

func failure(err error) error {
	var oleErr *ole.OleError
	if errors.As(err, &oleErr) && int32(oleErr.Code()) >= 0 {
		return nil
	}
	return err
}

func appKey(pid uint32) string {
	path := processPath(pid)
	if path == "" {
		return fmt.Sprintf("pid:%d", pid)
	}
	return strings.ToLower(path)
}

func describe(pid uint32, sessionName string) (string, image.Image) {
	process, ok := processName(pid)
	if !ok {
		if blank(sessionName) {
			return fmt.Sprintf("PID %d", pid), nil
		}
		return sessionName, nil
	}

	var name string
	var icon image.Image
	if path := processPath(pid); path != "" {
		name = fileDescription(path)
		icon = associatedIcon(path)
	}

	if blank(name) || genericName(name) {
		if !blank(sessionName) && !strings.HasPrefix(sessionName, "@") {
			name = sessionName
		} else if title := windowTitle(pid); title != "" {
			name = title
		} else if blank(name) {
			name = cleanProcessName(process)
		}
	}
	return name, icon
}

func blank(s string) bool {
	return strings.TrimSpace(s) == ""
}

func genericName(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "java") || strings.Contains(name, "openjdk")
}

func cleanProcessName(name string) string {
	for _, suffix := range []string{"-Win64-Shipping", "-Shipping", "-Win64"} {
		if len(name) >= len(suffix) && strings.EqualFold(name[len(name)-len(suffix):], suffix) {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}

func processPath(pid uint32) string {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(process)
	path := make([]uint16, 1024)
	n := uint32(len(path))
	if windows.QueryFullProcessImageName(process, 0, &path[0], &n) != nil {
		return ""
	}
	return windows.UTF16ToString(path[:n])
}

func processName(pid uint32) (string, bool) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(snapshot)
	process := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err := windows.Process32First(snapshot, &process); err == nil; err = windows.Process32Next(snapshot, &process) {
		if process.ProcessID != pid {
			continue
		}
		name := windows.UTF16ToString(process.ExeFile[:])
		if len(name) >= 4 && strings.EqualFold(name[len(name)-4:], ".exe") {
			name = name[:len(name)-4]
		}
		return name, true
	}
	return "", false
}

var mainWindow struct {
	pid  uint32
	hwnd windows.HWND
}

var findMainWindow = windows.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)
	if pid == mainWindow.pid && getWindow(hwnd, gwOwner) == 0 && windows.IsWindowVisible(hwnd) {
		mainWindow.hwnd = hwnd
		return 0
	}
	return 1
})

func windowTitle(pid uint32) string {
	mainWindow.pid, mainWindow.hwnd = pid, 0
	windows.EnumWindows(findMainWindow, nil)
	n := getWindowTextLength(mainWindow.hwnd)
	if mainWindow.hwnd == 0 || n == 0 {
		return ""
	}
	text := make([]uint16, n+1)
	getWindowText(mainWindow.hwnd, &text[0], n+1)
	return strings.TrimSpace(windows.UTF16ToString(text))
}

func fileDescription(path string) string {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 {
		return ""
	}
	info := make([]byte, size)
	block := unsafe.Pointer(&info[0])
	if windows.GetFileVersionInfo(path, 0, size, block) != nil {
		return ""
	}

	language := uint32(0x040904E4)
	var translation *[2]uint16
	var n uint32
	if windows.VerQueryValue(block, `\VarFileInfo\Translation`, unsafe.Pointer(&translation), &n) == nil && n >= 4 {
		language = uint32(translation[0])<<16 | uint32(translation[1])
	}

	query := func(language uint32, field string) string {
		var value *uint16
		var n uint32
		if windows.VerQueryValue(block, fmt.Sprintf(`\StringFileInfo\%08X\%s`, language, field), unsafe.Pointer(&value), &n) != nil || n == 0 {
			return ""
		}
		return windows.UTF16ToString(unsafe.Slice(value, n))
	}

	var description string
	for i, candidate := range []uint32{language, 0x040904B0, 0x040904E4, 0x04090000} {
		if i > 0 && candidate == language {
			continue
		}
		description = query(candidate, "FileDescription")
		if query(candidate, "FileVersion") != "" {
			break
		}
	}
	return description
}

func associatedIcon(path string) image.Image {
	if strings.HasPrefix(path, `\\`) || len(path) >= windows.MAX_PATH {
		return nil
	}
	buffer := make([]uint16, windows.MAX_PATH)
	copy(buffer, windows.StringToUTF16(path))
	var index uint16
	icon := extractAssociatedIcon(0, &buffer[0], &index)
	if icon == 0 {
		return nil
	}
	defer destroyIcon(icon)

	var info iconInfo
	if !getIconInfo(icon, &info) {
		return nil
	}
	defer deleteObject(info.mask)
	if info.color == 0 {
		return nil
	}
	defer deleteObject(info.color)

	var bmp bitmap
	getObject(info.color, int32(unsafe.Sizeof(bmp)), unsafe.Pointer(&bmp))
	if bmp.width <= 0 || bmp.height <= 0 {
		return nil
	}
	color := bitmapPixels(info.color, bmp.width, bmp.height)
	mask := bitmapPixels(info.mask, bmp.width, bmp.height)

	alpha := false
	for i := 3; i < len(color); i += 4 {
		alpha = alpha || color[i] != 0
	}

	img := image.NewNRGBA(image.Rect(0, 0, int(bmp.width), int(bmp.height)))
	for i := 0; i < len(color); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = color[i+2], color[i+1], color[i], color[i+3]
		if !alpha && mask[i] == 0 {
			img.Pix[i+3] = 0xFF
		}
	}
	return img
}

func bitmapPixels(bmp windows.Handle, width, height int32) []byte {
	pixels := make([]byte, width*height*4)
	info := bitmapInfo{width: width, height: -height, planes: 1, bitCount: 32}
	info.size = uint32(unsafe.Sizeof(info) - unsafe.Sizeof(info.colors))
	dc := createCompatibleDC(0)
	defer deleteDC(dc)
	getDIBits(dc, bmp, 0, uint32(height), &pixels[0], &info, 0)
	return pixels
}
