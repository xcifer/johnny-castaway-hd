# Johnny Castaway HD

A from-scratch engine for the 1992 Sierra/Dynamix screensaver *Screen Antics: Johnny Castaway*,
built as a native Windows 10/11 screensaver with optional HD art.

**Status:** in development. It plays the story with the original's sounds and understands the
Windows screensaver arguments. Settings, the preview and HD art are still to come.

## You need your own copy of the original

This repository contains **no** original game files or art. The engine reads the files from a
copy of *Screen Antics: Johnny Castaway* that you own:

- `SCRANTIC.SCR`
- `RESOURCE.MAP`
- `RESOURCE.001`

The screensaver looks for them next to `JohnnyCastawayHD.scr`, then in a `gamedata` folder next
to it. For development, put them in a `gamedata/` folder in the repo root, which is also searched
when the program is run from there. Git ignores this folder.

**Only have the install floppy?** On the floppy these files are compressed (`SCRANTIC.SC$`,
`RESOURCE.00$`). Copy everything from the floppy (or out of a disk image, e.g. with 7-Zip)
into `gamedata/floppy/` and run:

```
go run ./cmd/jcunpack
```

It decompresses the files, checks them against known MD5 hashes and writes them to
`gamedata/`. See [docs/files.md](docs/files.md).

## Building

Requires [Go](https://go.dev/dl/) 1.27 or newer. No C compiler is needed.

```
go build -o bin/jchd.exe ./cmd/jchd
```

The same program built as `JohnnyCastawayHD.scr` is the screensaver:

```
go build -ldflags=-H=windowsgui -o bin/JohnnyCastawayHD.scr ./cmd/jchd
```

`-H=windowsgui` keeps Windows from opening a console window when it starts the screensaver; the
price is that its log goes nowhere, so use `jchd.exe` to see it.

Windows starts a screensaver with `/s` to run it, `/c` for its settings and `/p` for the little
preview in Screen Saver Settings; with no arguments it shows the settings. The settings (sound,
picture, monitors and the game files' folder) are kept in `%APPDATA%\JohnnyCastawayHD\settings.json`;
options given to `jchd.exe` override them. The preview shows the original's title picture.

Run `bin/jchd.exe /s` (fullscreen; any key, mouse click or moving the mouse exits). It plays the story: scene after
scene on the island, with Johnny walking from one to the next. Night, holidays (Halloween
29-31 October, St Patrick's Day 15-17 March, Christmas 23-25 December, New Year 29 December
to 1 January) and the raft Johnny builds follow the clock and the day of the story, which
moves on once for each date the screensaver runs and is kept in
`%APPDATA%\JohnnyCastawayHD\progress.json`. See [docs/story.md](docs/story.md).

For development it also takes these options instead of `/s`, such as `bin/jchd.exe -windowed -debug`.
Without `-windowed`, it runs fullscreen and exits as with `/s`; in a window, moving the mouse
doesn't exit.

| Option | Effect |
|---|---|
| `-windowed` | Run in a 1280×720 window |
| `-debug` | Keys control the display: F1 info overlay, F2 scaling filter, Esc exits |
| `-filter sharp\|nearest\|smooth` | How the original art is scaled up (default `sharp`) |
| `-data DIR` | Game data folder (default: searched as described above) |
| `-seed N` | Fixed random seed, for repeatable runs |
| `-shot FILE.png -shotsize 3840x2160 -shotticks N` | Render one frame at any size after N ticks, save it and exit |
| `-ttm NAME` or `-ttm NAME:TAG,TAG` | Play one animation script, or some of its clips, over and over (see [docs/ttm-opcodes.md](docs/ttm-opcodes.md)) |
| `-ads NAME:SCENE` | Play one scene of a scene script over and over (see [docs/ads-opcodes.md](docs/ads-opcodes.md)) |
| `-left` | Move the island to the left, as some scenes do (for `-ttm` and `-ads`) |
| `-day N` | Play day N (1-11) of the story instead of the saved day, without saving |
| `-date "2026-12-24 15:30"` | Pretend the clock started at this time, for night and holidays, without saving |
| `-mute` | Play without sound |
| `-frames N` | Exit after N ticks (64 a second) and log the measured tick rate |
| `-monitor N` | Run fullscreen on monitor N (0, the default, is the main one) |
| `-follow PID` | Exit when the process with this ID ends |
| `-blank` | Show only black |

With more than one monitor, `/s` starts a copy of itself on each of the others with
`-monitor N -follow PID`: every monitor gets its own island and story, and only the main one plays
sound. When any of them ends, all do. If the settings ask for Johnny on the main monitor only, the
others get `-blank`.

The picture is always 16:9: the original 4:3 scene is centered and the sea extends to the sides.
On screens with another shape it gets black bars. The game clock is described in
[docs/timing.md](docs/timing.md).

## Inspecting the game data

```
go run ./cmd/jcdump
```

Extracts every sprite, background, script and sound into `gamedata/dump/`. Open
`gamedata/dump/index.html` in a browser to browse them. Format notes: [docs/formats.md](docs/formats.md).

Tests that need the game data skip when `gamedata/` is empty: `go test ./...`

## License

GPL-3.0. See [LICENSE](LICENSE) and [CREDITS.md](CREDITS.md).

*Johnny Castaway* and its art, sounds and scripts are the property of their rights holders.
This is an unofficial fan project, not affiliated with or endorsed by Sierra, Dynamix,
Activision or Microsoft.
