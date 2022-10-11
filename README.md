# x-ipc-viewer

IP camera viewer for X11.

# Features

- [mpv](https://mpv.io) with hardware decoding and low latency profile.
- Main and sub stream.
- Streams restart when they are out of sync.
- Window layouts.
  - Auto grid.
  - Manual placement.
- Fullscreen view.

# Key Bindings

| Key | Mouse          | Action                 |
| --- | -------------- | ---------------------- |
| q   |                | Quit                   |
| 1-9 | 2 x Left Click | Toggle Fullscreen View |
| 0   |                | Activate Normal View   |

# Config

Located at `~/.x-ipc-viewer.yml`.

Keys are NOT case sensitive.

```yaml
# Keep streams playing when they are not in view.
Background: false

# Layout for windows, [auto, manual]
Layout: auto

# Manual layout for windows, 'Layout' must be 'manual'.
# Define x, y, w (width), and h (height) as ratios of the full width and height.
# Ratios can be fractions or numbers.
# Coordinates start from the top left corner.
LayoutManual:
  - X: 0
    Y: 0
    W: .5
    H: 1
  - X: 0.5
    Y: 0
    W: 1/2
    H: 1/2
  - X: 1/2
    Y: 1/2
    W: 1/2
    H: 1/2

# Mpv player configuration.
Player:
  GPU: auto # The hardware decoding api to use. (--hwdec=<api>)
  LowLatency: true # Enable low-latency profile and disable cache.
  Flags: [] # Mpv flags.

# List of IP camera windows.
Windows:
  - Main: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=0 # Main stream used in fullscreen and/or normal view.
    Sub: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=1 # Sub stream used in normal view. (optional)
    LowLatency: true
    Flags: []
```

# Setup

This guide is for headless Debian 11 systems. Restart after finishing the guide.

## Installation

```
sudo apt install xserver-xorg xinit mpv
```

Create the directory `~/bin/`.

[Download x-ipc-viewer](https://github.com/ItsNotGoodName/x-ipc-viewer/releases) and place it in `~/bin/`.

## Start On Login

Create file at `~/.xinitrc` with the following content.

```sh
#!/bin/sh
[ -f /etc/xprofile ] && . /etc/xprofile
[ -f ~/.xprofile ] && . ~/.xprofile
exec x-ipc-viewer
```

Add the following content to the end of `~/.profile`.

```sh
if systemctl -q is-active graphical.target && [[ ! $DISPLAY && $XDG_VTNR -eq 1 ]]; then
  exec startx
fi
```

## Automatic Login

Run `sudo systemctl edit getty@tty1` and add the following block to it.
`username` should be changed to the user who will run the program.

```ini
[Service]
ExecStart=
ExecStart=-/sbin/agetty -o '-p -f -- \\u' --noclear --autologin username %I $TERM
```

## Enable Audio (Optional)

```shell
sudo apt install pulseaudio
```

## Hide Mouse When Not Moving (Optional)

```shell
sudo apt install unclutter
```

Add the following content to `~/.xprofile`.

```sh
unclutter &
```

# To Do

- ~~Add more layouts.~~
- ~~Add configurable [mpv](https://mpv.io) flags for each window.~~
- Add left click to focus window.
- Add multi-monitor support.
- Share audio between windows.
- Make switching between main and sub stream more seamless.
- Zooming.
