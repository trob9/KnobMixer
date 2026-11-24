package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var scale float64

func px(dips float64) float32 {
	return float32(math.RoundToEven(dips * scale))
}

type fontKey struct {
	size     float64
	semibold bool
}

type animation struct {
	fromOpacity, toOpacity float64
	fromSlide, toSlide     float64
	start                  time.Time
	duration               time.Duration
	done                   func()
}

type overlay struct {
	hwnd     windows.HWND
	dc       windows.Handle
	bitmap   windows.Handle
	measurer uintptr
	format   uintptr
	fonts    map[fontKey]uintptr
	pos      point
	size     size
	opacity  float64
	slide    float64
	shown    bool
	anim     animation
}

func newOverlay(class *uint16, instance windows.Handle) (*overlay, error) {
	hwnd, err := createWindowEx(wsExLayered|wsExTopmost|wsExNoActivate|wsExToolWindow|wsExTransparent, class, nil, wsPopup, 0, 0, 0, 0, 0, 0, instance, nil)
	if err != nil {
		return nil, err
	}
	o := &overlay{hwnd: hwnd, dc: createCompatibleDC(0), fonts: map[fontKey]uintptr{}}
	gdipCreateFromHDC(createCompatibleDC(0), &o.measurer)
	gdipSetTextRenderingHint(o.measurer, textAntiAliasGridFit)

	var typographic uintptr
	var flags int32
	gdipStringFormatGetGenericTypographic(&typographic)
	gdipCloneStringFormat(typographic, &o.format)
	gdipGetStringFormatFlags(o.format, &flags)
	gdipSetStringFormatFlags(o.format, flags|formatNoWrap)
	gdipSetStringFormatTrimming(o.format, trimmingEllipsisChar)
	return o, nil
}

func (o *overlay) font(size float64, semibold bool) uintptr {
	key := fontKey{size, semibold}
	if font, ok := o.fonts[key]; ok {
		return font
	}
	names := []string{"Segoe UI Variable Text", "Segoe UI"}
	if semibold {
		names = []string{"Segoe UI Variable Text Semibold", "Segoe UI Semibold"}
	}
	var family, font uintptr
	for _, name := range names {
		if gdipCreateFontFamilyFromName(utf16z(name), 0, &family) == 0 {
			break
		}
	}
	gdipCreateFont(family, f32(float32(size*scale)), 0, unitPixel, &font)
	o.fonts[key] = font
	return font
}

func (o *overlay) lineHeight(size float64) float32 {
	var height float32
	gdipGetFontHeight(o.font(size, false), o.measurer, &height)
	return float32(math.Round(float64(height)))
}

func (o *overlay) textWidth(text string, font uintptr) float32 {
	units := utf16.Encode([]rune(text))
	var bounds rectF
	gdipMeasureString(o.measurer, &units[0], int32(len(units)), font, &rectF{}, o.format, &bounds, nil, nil)
	return float32(math.Ceil(float64(bounds.w)))
}

type canvas struct {
	graphics uintptr
	format   uintptr
}

func roundRect(path uintptr, x, y, w, h, radius float32) {
	radius = min(radius, w/2, h/2)
	if radius <= 0 {
		gdipAddPathRectangle(path, f32(x), f32(y), f32(w), f32(h))
		return
	}
	d := 2 * radius
	gdipAddPathArc(path, f32(x), f32(y), f32(d), f32(d), f32(180), f32(90))
	gdipAddPathArc(path, f32(x+w-d), f32(y), f32(d), f32(d), f32(270), f32(90))
	gdipAddPathArc(path, f32(x+w-d), f32(y+h-d), f32(d), f32(d), f32(0), f32(90))
	gdipAddPathArc(path, f32(x), f32(y+h-d), f32(d), f32(d), f32(90), f32(90))
	gdipClosePathFigure(path)
}

func (c canvas) fill(argb uint32, shape func(path uintptr)) {
	if argb>>24 == 0 {
		return
	}
	var path, brush uintptr
	gdipCreatePath(0, &path)
	shape(path)
	gdipCreateSolidFill(argb, &brush)
	gdipFillPath(c.graphics, brush, path)
	gdipDeleteBrush(brush)
	gdipDeletePath(path)
}

func (c canvas) box(background, border uint32, x, y, w, h, radius, thickness float32) {
	if w <= 0 || h <= 0 {
		return
	}
	inner := func(path uintptr) {
		roundRect(path, x+thickness, y+thickness, w-2*thickness, h-2*thickness, radius-thickness/2)
	}
	c.fill(border, func(path uintptr) {
		roundRect(path, x, y, w, h, radius+thickness/2)
		inner(path)
	})
	c.fill(background, inner)
}

func (c canvas) text(text string, font uintptr, argb uint32, x, y, w float32, align int32) {
	units := utf16.Encode([]rune(text))
	if len(units) == 0 {
		return
	}
	var brush uintptr
	gdipCreateSolidFill(argb, &brush)
	gdipSetStringFormatAlign(c.format, align)
	gdipDrawString(c.graphics, &units[0], int32(len(units)), font, &rectF{x: x, y: y, w: w}, c.format, brush)
	gdipDeleteBrush(brush)
}

func (c canvas) image(img image.Image, x, y, size float32, opacity float64) {
	bounds := img.Bounds()
	pixels := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for col := bounds.Min.X; col < bounds.Max.X; col++ {
			p := color.NRGBAModel.Convert(img.At(col, row)).(color.NRGBA)
			pixels = append(pixels, p.B, p.G, p.R, byte(math.Round(float64(p.A)*opacity)))
		}
	}
	var bmp uintptr
	gdipCreateBitmapFromScan0(int32(bounds.Dx()), int32(bounds.Dy()), int32(bounds.Dx()*4), pixelFormat32bppARGB, unsafe.Pointer(&pixels[0]), &bmp)
	gdipDrawImageRectI(c.graphics, bmp, int32(x), int32(y), int32(size), int32(size))
	gdipDisposeImage(bmp)
	runtime.KeepAlive(pixels)
}

type hint struct {
	key, action           string
	keyWidth, actionWidth float32
}

func (o *overlay) render(entries []*entry, selected int, m mode) error {
	drawn, err := o.draw(entries, selected, m)
	if err != nil {
		return err
	}
	o.size = drawn
	o.pos = point{}
	monitor := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if getMonitorInfo(monitorFromWindow(windows.GetForegroundWindow(), monitorToPrimary), &monitor) {
		o.pos.x = monitor.monitor.Left + (monitor.monitor.Right-monitor.monitor.Left-o.size.cx)/2
		o.pos.y = monitor.monitor.Top + int32(48*scale)
	}
	o.present()
	setWindowPos(o.hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)

	if !o.shown {
		o.shown = true
		o.animate(1, 0, 150*time.Millisecond, nil)
	}
	return nil
}

func (o *overlay) draw(entries []*entry, selected int, m mode) (size, error) {
	hints := []hint{{key: "Turn", action: "Choose"}, {key: "Click", action: "Adjust"}, {key: "Backspace", action: "Close"}}
	if m != selecting {
		hints = []hint{{key: "Turn", action: "Volume"}, {key: "Backspace", action: "Back"}, {key: "Click", action: "Done"}}
	}

	percentSize := 16.0
	if m == adjusting {
		percentSize = 20
	}

	border := px(1)
	padding := px(10)
	tileBorder := px(1.5)
	tileMargin := px(2)
	tileWidth := px(108)
	tileHeight := 2*tileBorder + 2*px(12) + px(32) + px(8) + o.lineHeight(12) + px(2) + o.lineHeight(percentSize) + px(8) + px(4)
	contentWidth := tileWidth - 2*tileBorder - 2*px(10)
	iconSize := px(28)
	capHeight := 2*border + px(1) + o.lineHeight(11) + px(2)

	var hintsWidth float32
	for i := range hints {
		hints[i].keyWidth = o.textWidth(hints[i].key, o.font(11, true))
		hints[i].actionWidth = o.textWidth(hints[i].action, o.font(11, false))
		hintsWidth += px(8) + border + px(6) + hints[i].keyWidth + px(6) + border + px(6) + hints[i].actionWidth + px(8)
	}

	innerWidth := max(float32(len(entries))*(tileWidth+2*tileMargin), hintsWidth)
	width := border + padding + innerWidth + padding + border
	height := border + padding + tileMargin + tileHeight + tileMargin + px(10) + px(1) + px(10) + capHeight + px(12) + border

	var bits unsafe.Pointer
	info := bitmapInfo{width: int32(width), height: -int32(height), planes: 1, bitCount: 32}
	info.size = uint32(unsafe.Sizeof(info) - unsafe.Sizeof(info.colors))
	bitmap := createDIBSection(o.dc, &info, 0, &bits, 0, 0)
	if bitmap == 0 {
		return size{}, errors.New("could not create overlay bitmap")
	}
	selectObject(o.dc, bitmap)
	deleteObject(o.bitmap)
	o.bitmap = bitmap

	var surfaceBitmap uintptr
	c := canvas{format: o.format}
	gdipCreateBitmapFromScan0(int32(width), int32(height), int32(width)*4, pixelFormat32bppPARGB, bits, &surfaceBitmap)
	defer gdipDisposeImage(surfaceBitmap)
	if status := gdipGetImageGraphicsContext(surfaceBitmap, &c.graphics); status != 0 {
		return size{}, fmt.Errorf("could not draw overlay: gdiplus status %d", status)
	}
	defer gdipDeleteGraphics(c.graphics)
	gdipSetSmoothingMode(c.graphics, smoothingAntiAlias)
	gdipSetPixelOffsetMode(c.graphics, pixelOffsetHalf)
	gdipSetTextRenderingHint(c.graphics, textAntiAliasGridFit)
	gdipSetInterpolationMode(c.graphics, interpolationBicubic)

	c.box(surface, hairline, 0, 0, width, height, px(14), border)

	for i, e := range entries {
		volume, err := e.volume()
		if err != nil {
			return size{}, err
		}
		muted, err := e.muted()
		if err != nil {
			return size{}, err
		}

		isSelected := i == selected
		isAdjusting := isSelected && m == adjusting

		var foreground, bar, background, edge uint32 = textSecondary, fillIdle, 0, 0
		switch {
		case isAdjusting:
			foreground, bar, background, edge = textPrimary, accent, accentTint, accent
		case isSelected:
			foreground, bar, background, edge = textPrimary, textPrimary, surfaceRaised, hairlineStrong
		}

		x := border + padding + float32(i)*(tileWidth+2*tileMargin) + tileMargin
		y := border + padding + tileMargin
		c.box(background, edge, x, y, tileWidth, tileHeight, px(10), tileBorder)

		x += tileBorder + px(10)
		y += tileBorder + px(12)

		iconX := x + float32(math.RoundToEven(float64(contentWidth-iconSize)/2))
		iconY := y + float32(math.RoundToEven(float64(px(32)-iconSize)/2))
		switch {
		case e.endpoint != nil && muted:
			c.image(speakerGlyph(int(iconSize), mutedRed, volume, true), iconX, iconY, iconSize, 1)
		case e.endpoint != nil:
			c.image(speakerGlyph(int(iconSize), foreground, volume, false), iconX, iconY, iconSize, 1)
		case e.icon != nil && isSelected:
			c.image(e.icon, iconX, iconY, iconSize, 1)
		case e.icon != nil:
			c.image(e.icon, iconX, iconY, iconSize, 0.75)
		default:
			c.image(appGlyph(int(iconSize), foreground), iconX, iconY, iconSize, 1)
		}
		y += px(32) + px(8)

		c.text(e.name, o.font(12, isSelected), foreground, x, y, contentWidth, alignCenter)
		y += o.lineHeight(12) + px(2)

		percent := fmt.Sprintf("%d%%", int(math.RoundToEven(float64(volume*100))))
		percentColor := foreground
		tileFontSize := 16.0
		if isAdjusting {
			percentColor = accent
			tileFontSize = 20
		}
		if muted {
			percent = "Muted"
			percentColor = mutedRed
		}
		c.text(percent, o.font(tileFontSize, true), percentColor, x, y, contentWidth, alignCenter)
		y += o.lineHeight(tileFontSize) + px(8)

		trackWidth := contentWidth - 2*px(2)
		filled := float32(math.RoundToEven(float64(trackWidth * min(max(volume, 0), 1))))
		c.box(track, 0, x+px(2), y, trackWidth, px(4), px(2), 0)
		c.box(bar, 0, x+px(2), y, filled, px(4), px(2), 0)
	}

	y := border + padding + tileMargin + tileHeight + tileMargin + px(10)
	c.box(hairline, 0, border+padding+px(4), y, innerWidth-2*px(4), px(1), 0, 0)

	y += px(1) + px(10)
	x := border + padding + float32(math.RoundToEven(float64(innerWidth-hintsWidth)/2))
	textY := y + float32(math.RoundToEven(float64(capHeight-o.lineHeight(11))/2))
	for _, h := range hints {
		x += px(8)
		capWidth := border + px(6) + h.keyWidth + px(6) + border
		c.box(surfaceRaised, hairlineStrong, x, y, capWidth, capHeight, px(4), border)
		c.text(h.key, o.font(11, true), textPrimary, x+border+px(6), y+border+px(1), 0, alignNear)
		x += capWidth + px(6)
		c.text(h.action, o.font(11, false), textSecondary, x, textY, 0, alignNear)
		x += h.actionWidth + px(8)
	}

	return size{int32(width), int32(height)}, nil
}

func (o *overlay) hide() {
	if !o.shown {
		return
	}
	o.shown = false
	o.animate(0, 4, 110*time.Millisecond, func() { showWindow(o.hwnd, swHide) })
}

func (o *overlay) present() {
	pos := point{o.pos.x, o.pos.y + int32(math.Round(o.slide*scale))}
	blend := blendFunction{alpha: byte(math.Round(o.opacity * 255)), format: acSrcAlpha}
	updateLayeredWindow(o.hwnd, 0, &pos, &o.size, o.dc, &point{}, 0, &blend, ulwAlpha)
}

func (o *overlay) animate(opacity, slide float64, duration time.Duration, done func()) {
	var enabled int32
	systemParametersInfo(spiGetClientAreaAnimation, 0, unsafe.Pointer(&enabled), 0)
	if enabled == 0 {
		killTimer(window, timerAnimate)
		o.opacity, o.slide = opacity, 0
		o.present()
		if done != nil {
			done()
		}
		return
	}
	o.anim = animation{o.opacity, opacity, o.slide, slide, time.Now(), duration, done}
	setTimer(window, timerAnimate, 10, 0)
}

func (o *overlay) tick() {
	a := o.anim
	t := min(float64(time.Since(a.start))/float64(a.duration), 1)
	eased := 1 - math.Pow(1-t, 3)
	o.opacity = a.fromOpacity + (a.toOpacity-a.fromOpacity)*eased
	o.slide = a.fromSlide + (a.toSlide-a.fromSlide)*eased
	o.present()
	if t < 1 {
		return
	}
	killTimer(window, timerAnimate)
	if a.done != nil {
		a.done()
	}
}
