# x-ipc-viewer

[![GitHub](https://img.shields.io/github/license/itsnotgoodname/x-ipc-viewer)](./LICENSE)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/itsnotgoodname/x-ipc-viewer)](https://github.com/ItsNotGoodName/x-ipc-viewer/tags)
[![GitHub last commit](https://img.shields.io/github/last-commit/itsnotgoodname/x-ipc-viewer)](https://github.com/ItsNotGoodName/x-ipc-viewer)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/itsnotgoodname/x-ipc-viewer)](./go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/ItsNotGoodName/x-ipc-viewer)](https://goreportcard.com/report/github.com/ItsNotGoodName/x-ipc-viewer)

IP camera viewer for X11.

# Features

- [mpv](https://mpv.io) as the video player.
- Main and sub stream.
- Layout view.
  - Auto grid.
  - Manual placement.
- Fullscreen view.

# Key Bindings

| Key | Mouse          | Action                 |
| --- | -------------- | ---------------------- |
| q   |                | Quit                   |
| 1-9 | 2 x Left Click | Toggle Fullscreen View |
| 0   |                | Activate Layout View   |

# Configuration

Located at `~/.x-ipc-viewer.yml`.

Keys are NOT case sensitive.

```yaml
# Keep streams playing when they are not in view.
Background: false

# Layout for windows. [auto, manual]
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
  Flags: [] # Mpv flags.

# List of windows.
Windows:
  - Main: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=0 # Main stream used in fullscreen and/or normal view.
    Sub: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=1 # Sub stream used in normal view. (optional)
    LowLatency: true # Enable low-latency profile and disable cache. Should be used for streams from IP cameras.
  - Name: Foo video # Name for logging purposes.
    Main: /tmp/foo.mp4
    Flags: # Extra mpv flags.
      - --no-keepaspect # Stretch window.
      - --glsl-shader=/tmp/nonlinear_stretch.glsl # https://gist.github.com/sarahzrf/c9909aee70e3656895820f20ac395956
```

# Setup

This guide is for headless Debian 11 systems. Restart after finishing the guide.

## Installation

Run the following command.

```
sudo apt install xserver-xorg xinit mpv
```

Create the directory `~/.local/bin/`.

[Download](https://github.com/ItsNotGoodName/x-ipc-viewer/releases/latest) the binary and place it in `~/.local/bin/`.

## Start On Login

Create file at `~/.xinitrc` with the following content.

```sh
#!/bin/sh
[ -f /etc/xprofile ] && . /etc/xprofile
[ -f ~/.xprofile ] && . ~/.xprofile
exec x-ipc-viewer --config-watch-exit
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

Run the following command.

```shell
sudo apt install pulseaudio
```

## Hide Mouse When Not Moving (Optional)

Run the following command.

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
- Mute window with unfocus and focus events.
- Zooming.
- Add multi-monitor support.
- Share audio between windows.
- Make switching between main and sub stream more seamless.
