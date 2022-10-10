# x-ipc-viewer

IP camera viewer for X11.

# Features

- Handle main and sub stream.
- Auto grid layout.
- Fullscreen view.
- Uses [mpv](https://mpv.io) with hardware decoding and low latency profile.
- Streams restart when they are out of sync.

# Key Bindings

| Key | Mouse          | Action               |
| --- | -------------- | -------------------- |
| q   |                | Quit                 |
| 1-9 | 2 x Left Click | Toggle Fullscreen    |
| 0   |                | Activate Normal View |

# Config

Located at `~/.x-ipc-viewer.yml`.

```yaml
background: false # Keep streams playing when they are not in view.

windows: # List of IP camera windows.
  - main: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=0 # Main stream used in fullscreen and/or normal view.
    sub: rtsp://admin:password@192.168.1.108:554/cam/realmonitor?channel=1&subtype=1 # Sub stream used in normal view. (optional)
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

- Add more layouts.
- Add configurable [mpv](https://mpv.io) flags for each window.
- Share audio between windows.
- Add left click to focus window.
- Add multi-monitor support.
- Zooming.
- Make switching between main and sub stream more seamless.
