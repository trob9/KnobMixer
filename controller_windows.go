package main

import "unsafe"

const (
	vkBack   = 0x08
	vkEscape = 0x1B
	vkMute   = 0xAD
	vkDown   = 0xAE
	vkUp     = 0xAF

	step = 0.02
)

type mode int

const (
	closed mode = iota
	selecting
	adjusting
)

type controller struct {
	audio   audio
	overlay *overlay
	eaten   map[uint32]bool
	mode    mode
	entries []*entry
	index   int
}

func (c *controller) hook(code int32, wParam uintptr, key *kbdllHookStruct) uintptr {
	if code >= 0 {
		down := wParam == wmKeyDown || wParam == wmSysKeyDown
		up := wParam == wmKeyUp || wParam == wmSysKeyUp
		if (down || up) && c.onKey(key.vkCode, down) {
			return 1
		}
	}
	return callNextHookEx(0, code, wParam, uintptr(unsafe.Pointer(key)))
}

func (c *controller) onKey(vk uint32, down bool) bool {
	if !down {
		eaten := c.eaten[vk]
		delete(c.eaten, vk)
		return eaten
	}

	repeat := c.eaten[vk]

	if c.mode == closed {
		if vk != vkMute {
			return false
		}
		c.mode = selecting
		c.eaten[vk] = true
		post(c.open)
		return true
	}

	m := c.mode
	switch vk {
	case vkUp:
		post(func() error { return c.turn(1, m) })
	case vkDown:
		post(func() error { return c.turn(-1, m) })
	case vkMute:
		if !repeat {
			c.mode = closed
			if m == selecting {
				c.mode = adjusting
			}
			next := c.mode
			post(func() error { return c.apply(next) })
		}
	case vkBack:
		if !repeat {
			c.mode = closed
			if m == adjusting {
				c.mode = selecting
			}
			next := c.mode
			post(func() error { return c.apply(next) })
		}
	case vkEscape:
		c.mode = closed
		post(func() error { return c.apply(closed) })
	default:
		return false
	}
	c.eaten[vk] = true
	return true
}

func (c *controller) idle() {
	if c.mode == closed {
		killTimer(window, timerIdle)
		return
	}
	c.mode = closed
	post(func() error { return c.apply(closed) })
}

func (c *controller) open() error {
	entries, err := c.audio.refresh()
	if err != nil {
		return err
	}
	c.entries = entries
	c.index = 0
	return c.apply(selecting)
}

func (c *controller) turn(direction int, m mode) error {
	if len(c.entries) == 0 {
		return nil
	}
	switch m {
	case selecting:
		c.index = (c.index + direction + len(c.entries)) % len(c.entries)
	case adjusting:
		if err := c.entries[c.index].nudge(float32(direction) * step); err != nil {
			return err
		}
	}
	return c.apply(m)
}

func (c *controller) apply(m mode) error {
	killTimer(window, timerIdle)
	if m == closed {
		c.overlay.hide()
		c.audio.release()
		return nil
	}
	if err := c.overlay.render(c.entries, c.index, m); err != nil {
		return err
	}
	setTimer(window, timerIdle, 5000, 0)
	return nil
}
