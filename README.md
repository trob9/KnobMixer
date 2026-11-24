# KnobMixer

Control your Windows volume mixer with a single knob, even inside games.

Built for keyboards with a volume knob (Keychron, etc.). Turn it for global volume as normal. Click it to open a small overlay where you pick an app and turn the knob to change just that app's volume, so you can turn the game up and Discord down without leaving your game.

<img width="550" height="263" alt="image" src="https://github.com/user-attachments/assets/8cf03a43-6816-462b-bdb7-baaebf9a0c56" />


## How it works

| Action | What happens |
|---|---|
| Turn knob (overlay closed) | Normal global volume (Windows handles it) |
| Click knob | Open overlay: Master + every app playing sound |
| Turn knob | Choose which one to control |
| Click knob | Adjust the selected one (2% per notch) |
| Backspace | Back to choosing (or close, if already choosing) |
| Click knob (while adjusting) | Close overlay |
| Esc | Close overlay from anywhere |

The overlay opens at the top-centre of the monitor your active window is on, never steals focus, and ignores the mouse. It auto-closes after 5 seconds of inactivity.

## Requirements

- Windows 10/11 (x64)
- A knob that sends the standard media keys: **click = Volume Mute**, **turn = Volume Up/Down** (the default on most boards)

## Build and run

Needs [Go](https://go.dev/dl/). The result is a single exe with no runtime to install.

```
go build -ldflags -H=windowsgui -o dist\KnobMixer.exe
dist\KnobMixer.exe
```

It lives in the system tray (knob icon). Right-click it for **Start with Windows** and **Quit**.

## Limitations

- **Exclusive-fullscreen games** can hide any overlay. Borderless / windowed-fullscreen is fine.
- **Admin windows:** Windows blocks keyboard hooks while an elevated window has focus, so the knob won't respond then unless KnobMixer is also run as admin.
- The click no longer mutes, since it opens the overlay instead.
- Apps only appear once they have an audio stream open (Discord shows up when you are in a call or it is making sound).
- Errors are logged to `%LocalAppData%\KnobMixer\log.txt`.

## How it's built

- `main_windows.go`: startup, message loop and the queue the hook hands work to
- `controller_windows.go`: global low-level keyboard hook (works regardless of focus) and the click / turn / backspace state machine
- `audio_windows.go`: per-app volume via [go-wca](https://github.com/moutend/go-wca) (Windows Core Audio sessions)
- `overlay_windows.go`: the no-focus, always-on-top layered overlay, drawn with GDI+
- `tray_windows.go`: tray icon, menu and the Start with Windows registry entry
- `theme.go` / `icons.go`: design tokens (flat dark theme, one teal accent) and vector icons
- `syscall_windows.go`: Win32 declarations; `go generate` writes `zsyscall_windows.go` from them

## License

MIT
