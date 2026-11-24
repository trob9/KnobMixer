package main

import (
	"math"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

//go:generate go tool mkwinsyscall -output zsyscall_windows.go syscall_windows.go

const (
	wmNull       = 0x0000
	wmTimer      = 0x0113
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105
	wmRButtonUp  = 0x0205
	wmApp        = 0x8000

	wsPopup         = 0x80000000
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExTopmost     = 0x00000008
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000

	swHide           = 0
	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoActivate    = 0x0010
	swpShowWindow    = 0x0040
	hwndTopmost      = ^windows.HWND(0)
	gwOwner          = 4
	ulwAlpha         = 2
	acSrcAlpha       = 1
	whKeyboardLL     = 13
	monitorToPrimary = 1

	spiGetClientAreaAnimation = 0x1042

	nimAdd     = 0
	nimDelete  = 2
	nifMessage = 1
	nifIcon    = 2
	nifTip     = 4

	mfString       = 0x0000
	mfChecked      = 0x0008
	mfUnchecked    = 0x0000
	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100

	pixelFormat32bppARGB  = 0x0026200A
	pixelFormat32bppPARGB = 0x000E200B
	unitPixel             = 2
	smoothingAntiAlias    = 4
	pixelOffsetHalf       = 4
	textAntiAliasGridFit  = 3
	interpolationBicubic  = 7
	trimmingEllipsisChar  = 3
	formatNoWrap          = 0x1000
	alignNear             = 0
	alignCenter           = 1
)

type point struct{ x, y int32 }

type size struct{ cx, cy int32 }

type rectF struct{ x, y, w, h float32 }

type msg struct {
	hwnd    windows.HWND
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSm     windows.Handle
}

type monitorInfo struct {
	size    uint32
	monitor windows.Rect
	work    windows.Rect
	flags   uint32
}

type blendFunction struct{ op, flags, alpha, format byte }

type bitmapInfo struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
	colors        [3]uint32
}

type bitmap struct {
	kind       int32
	width      int32
	height     int32
	widthBytes int32
	planes     uint16
	bitsPixel  uint16
	bits       uintptr
}

type iconInfo struct {
	icon     int32
	xHotspot uint32
	yHotspot uint32
	mask     windows.Handle
	color    windows.Handle
}

type notifyIconData struct {
	size            uint32
	hwnd            windows.HWND
	id              uint32
	flags           uint32
	callbackMessage uint32
	icon            windows.Handle
	tip             [128]uint16
	state           uint32
	stateMask       uint32
	info            [256]uint16
	version         uint32
	infoTitle       [64]uint16
	infoFlags       uint32
	guid            windows.GUID
	balloonIcon     windows.Handle
}

type kbdllHookStruct struct {
	vkCode    uint32
	scanCode  uint32
	flags     uint32
	time      uint32
	extraInfo uintptr
}

type gdiplusStartupInput struct {
	version                  uint32
	debugCallback            uintptr
	suppressBackgroundThread int32
	suppressExternalCodecs   int32
}

func f32(v float32) uint32 { return math.Float32bits(v) }

func utf16z(s string) *uint16 {
	return &append(utf16.Encode([]rune(s)), 0)[0]
}

//sys	registerClassEx(class *wndClassEx) (atom uint16, err error) = user32.RegisterClassExW
//sys	createWindowEx(exStyle uint32, className *uint16, windowName *uint16, style uint32, x int32, y int32, width int32, height int32, parent windows.HWND, menu windows.Handle, instance windows.Handle, param unsafe.Pointer) (hwnd windows.HWND, err error) = user32.CreateWindowExW
//sys	defWindowProc(hwnd windows.HWND, message uint32, wParam uintptr, lParam uintptr) (ret uintptr) = user32.DefWindowProcW
//sys	getMessage(m *msg, hwnd windows.HWND, min uint32, max uint32) (ret int32) = user32.GetMessageW
//sys	dispatchMessage(m *msg) (ret uintptr) = user32.DispatchMessageW
//sys	postMessage(hwnd windows.HWND, message uint32, wParam uintptr, lParam uintptr) (err error) = user32.PostMessageW
//sys	postQuitMessage(code int32) = user32.PostQuitMessage
//sys	registerWindowMessage(name *uint16) (message uint32) = user32.RegisterWindowMessageW
//sys	setTimer(hwnd windows.HWND, id uintptr, elapse uint32, fn uintptr) (ret uintptr) = user32.SetTimer
//sys	killTimer(hwnd windows.HWND, id uintptr) (ok bool) = user32.KillTimer
//sys	setWindowsHookEx(id int32, fn uintptr, module windows.Handle, thread uint32) (hook windows.Handle, err error) = user32.SetWindowsHookExW
//sys	unhookWindowsHookEx(hook windows.Handle) (ok bool) = user32.UnhookWindowsHookEx
//sys	callNextHookEx(hook windows.Handle, code int32, wParam uintptr, lParam uintptr) (ret uintptr) = user32.CallNextHookEx
//sys	showWindow(hwnd windows.HWND, cmd int32) (visible bool) = user32.ShowWindow
//sys	setWindowPos(hwnd windows.HWND, after windows.HWND, x int32, y int32, cx int32, cy int32, flags uint32) (ok bool) = user32.SetWindowPos
//sys	updateLayeredWindow(hwnd windows.HWND, dst windows.Handle, pos *point, sz *size, src windows.Handle, srcPos *point, key uint32, blend *blendFunction, flags uint32) (ok bool) = user32.UpdateLayeredWindow
//sys	monitorFromWindow(hwnd windows.HWND, flags uint32) (monitor windows.Handle) = user32.MonitorFromWindow
//sys	getMonitorInfo(monitor windows.Handle, info *monitorInfo) (ok bool) = user32.GetMonitorInfoW
//sys	setProcessDPIAware() (ok bool) = user32.SetProcessDPIAware
//sys	getDpiForSystem() (dpi uint32) = user32.GetDpiForSystem
//sys	systemParametersInfo(action uint32, param uint32, value unsafe.Pointer, winIni uint32) (ok bool) = user32.SystemParametersInfoW
//sys	createPopupMenu() (menu windows.Handle) = user32.CreatePopupMenu
//sys	appendMenu(menu windows.Handle, flags uint32, id uintptr, item *uint16) (ok bool) = user32.AppendMenuW
//sys	checkMenuItem(menu windows.Handle, id uint32, check uint32) (previous int32) = user32.CheckMenuItem
//sys	trackPopupMenu(menu windows.Handle, flags uint32, x int32, y int32, reserved int32, hwnd windows.HWND, rect *windows.Rect) (cmd int32) = user32.TrackPopupMenu
//sys	getCursorPos(pt *point) (ok bool) = user32.GetCursorPos
//sys	setForegroundWindow(hwnd windows.HWND) (ok bool) = user32.SetForegroundWindow
//sys	getWindow(hwnd windows.HWND, cmd uint32) (window windows.HWND) = user32.GetWindow
//sys	getWindowTextLength(hwnd windows.HWND) (n int32) = user32.GetWindowTextLengthW
//sys	getWindowText(hwnd windows.HWND, text *uint16, max int32) (n int32) = user32.GetWindowTextW
//sys	getIconInfo(icon windows.Handle, info *iconInfo) (ok bool) = user32.GetIconInfo
//sys	destroyIcon(icon windows.Handle) (ok bool) = user32.DestroyIcon

//sys	createCompatibleDC(dc windows.Handle) (memory windows.Handle) = gdi32.CreateCompatibleDC
//sys	createDIBSection(dc windows.Handle, info *bitmapInfo, usage uint32, bits *unsafe.Pointer, section windows.Handle, offset uint32) (bmp windows.Handle) = gdi32.CreateDIBSection
//sys	deleteDC(dc windows.Handle) (ok bool) = gdi32.DeleteDC
//sys	selectObject(dc windows.Handle, object windows.Handle) (previous windows.Handle) = gdi32.SelectObject
//sys	deleteObject(object windows.Handle) (ok bool) = gdi32.DeleteObject
//sys	getObject(object windows.Handle, size int32, out unsafe.Pointer) (n int32) = gdi32.GetObjectW
//sys	getDIBits(dc windows.Handle, bmp windows.Handle, start uint32, lines uint32, bits *byte, info *bitmapInfo, usage uint32) (n int32) = gdi32.GetDIBits

//sys	shellNotifyIcon(message uint32, data *notifyIconData) (ok bool) = shell32.Shell_NotifyIconW
//sys	extractAssociatedIcon(instance windows.Handle, path *uint16, index *uint16) (icon windows.Handle) = shell32.ExtractAssociatedIconW

//sys	gdiplusStartup(token *uintptr, input *gdiplusStartupInput, output unsafe.Pointer) (status uint32) = gdiplus.GdiplusStartup
//sys	gdipCreateBitmapFromScan0(width int32, height int32, stride int32, format int32, scan0 unsafe.Pointer, bmp *uintptr) (status uint32) = gdiplus.GdipCreateBitmapFromScan0
//sys	gdipCreateHICONFromBitmap(bmp uintptr, icon *windows.Handle) (status uint32) = gdiplus.GdipCreateHICONFromBitmap
//sys	gdipDisposeImage(image uintptr) (status uint32) = gdiplus.GdipDisposeImage
//sys	gdipGetImageGraphicsContext(image uintptr, graphics *uintptr) (status uint32) = gdiplus.GdipGetImageGraphicsContext
//sys	gdipCreateFromHDC(dc windows.Handle, graphics *uintptr) (status uint32) = gdiplus.GdipCreateFromHDC
//sys	gdipDeleteGraphics(graphics uintptr) (status uint32) = gdiplus.GdipDeleteGraphics
//sys	gdipSetSmoothingMode(graphics uintptr, mode int32) (status uint32) = gdiplus.GdipSetSmoothingMode
//sys	gdipSetPixelOffsetMode(graphics uintptr, mode int32) (status uint32) = gdiplus.GdipSetPixelOffsetMode
//sys	gdipSetTextRenderingHint(graphics uintptr, hint int32) (status uint32) = gdiplus.GdipSetTextRenderingHint
//sys	gdipSetInterpolationMode(graphics uintptr, mode int32) (status uint32) = gdiplus.GdipSetInterpolationMode
//sys	gdipCreateSolidFill(argb uint32, brush *uintptr) (status uint32) = gdiplus.GdipCreateSolidFill
//sys	gdipDeleteBrush(brush uintptr) (status uint32) = gdiplus.GdipDeleteBrush
//sys	gdipFillEllipseI(graphics uintptr, brush uintptr, x int32, y int32, width int32, height int32) (status uint32) = gdiplus.GdipFillEllipseI
//sys	gdipFillRectangleI(graphics uintptr, brush uintptr, x int32, y int32, width int32, height int32) (status uint32) = gdiplus.GdipFillRectangleI
//sys	gdipCreatePath(fillMode int32, path *uintptr) (status uint32) = gdiplus.GdipCreatePath
//sys	gdipDeletePath(path uintptr) (status uint32) = gdiplus.GdipDeletePath
//sys	gdipAddPathArc(path uintptr, x uint32, y uint32, width uint32, height uint32, start uint32, sweep uint32) (status uint32) = gdiplus.GdipAddPathArc
//sys	gdipAddPathRectangle(path uintptr, x uint32, y uint32, width uint32, height uint32) (status uint32) = gdiplus.GdipAddPathRectangle
//sys	gdipClosePathFigure(path uintptr) (status uint32) = gdiplus.GdipClosePathFigure
//sys	gdipFillPath(graphics uintptr, brush uintptr, path uintptr) (status uint32) = gdiplus.GdipFillPath
//sys	gdipDrawImageRectI(graphics uintptr, image uintptr, x int32, y int32, width int32, height int32) (status uint32) = gdiplus.GdipDrawImageRectI
//sys	gdipCreateFontFamilyFromName(name *uint16, collection uintptr, family *uintptr) (status uint32) = gdiplus.GdipCreateFontFamilyFromName
//sys	gdipCreateFont(family uintptr, emSize uint32, style int32, unit int32, font *uintptr) (status uint32) = gdiplus.GdipCreateFont
//sys	gdipGetFontHeight(font uintptr, graphics uintptr, height *float32) (status uint32) = gdiplus.GdipGetFontHeight
//sys	gdipStringFormatGetGenericTypographic(format *uintptr) (status uint32) = gdiplus.GdipStringFormatGetGenericTypographic
//sys	gdipCloneStringFormat(format uintptr, clone *uintptr) (status uint32) = gdiplus.GdipCloneStringFormat
//sys	gdipGetStringFormatFlags(format uintptr, flags *int32) (status uint32) = gdiplus.GdipGetStringFormatFlags
//sys	gdipSetStringFormatFlags(format uintptr, flags int32) (status uint32) = gdiplus.GdipSetStringFormatFlags
//sys	gdipSetStringFormatAlign(format uintptr, align int32) (status uint32) = gdiplus.GdipSetStringFormatAlign
//sys	gdipSetStringFormatTrimming(format uintptr, trimming int32) (status uint32) = gdiplus.GdipSetStringFormatTrimming
//sys	gdipMeasureString(graphics uintptr, text *uint16, length int32, font uintptr, layout *rectF, format uintptr, bounds *rectF, codepoints *int32, lines *int32) (status uint32) = gdiplus.GdipMeasureString
//sys	gdipDrawString(graphics uintptr, text *uint16, length int32, font uintptr, layout *rectF, format uintptr, brush uintptr) (status uint32) = gdiplus.GdipDrawString
