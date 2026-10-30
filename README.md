# Johnny Castaway HD

A from-scratch remake of the 1992 Sierra/Dynamix screensaver *Screen Antics: Johnny Castaway*, as
a screensaver for Windows 10 and 11. Johnny lives out his story on his little island again:
fishing, building sandcastles, diving, jogging, with the visitors, the night, the holidays and the
raft he builds over the days. It plays the original's own animations, scenes and sounds, at the
original's pace, drawn sharp at any screen size up to 4K, in 16:9 with the sea extended to the
sides.

## You need your own copy of the original

This project contains **no** original game files or art. It plays the files from a copy of
*Screen Antics: Johnny Castaway* that you own:

- `SCRANTIC.SCR`
- `RESOURCE.MAP`
- `RESOURCE.001`

**Only have the install floppy?** On the floppy two of these files are compressed
(`SCRANTIC.SC$`, `RESOURCE.00$`). `jcunpack.exe`, which comes with the screensaver, unpacks them:
see step 2 below.

## Installing

1. Download `JohnnyCastawayHD-1.0.zip` from the [Releases](../../releases) page and unzip it into a
   folder of its own, for example `C:\Johnny Castaway HD`. Keep the screensaver in that folder.
2. Put the three original files next to `JohnnyCastawayHD.scr`, or in a folder called `gamedata`
   there.
   From the floppy: copy everything on it (or out of a disk image, e.g. with 7-Zip) into
   `gamedata\floppy` in that folder and run `jcunpack.exe`. It writes the three files to
   `gamedata`, after checking them against the known versions of the original.
3. Right-click `JohnnyCastawayHD.scr` and choose **Install**. Windows opens Screen Saver Settings
   with Johnny Castaway HD chosen; press **OK**.
4. **Settings...** in that window lets you choose:
   - whether Johnny makes sounds,
   - how the picture is scaled up: *Sharp* (recommended), *Blocky pixels* or *Smooth*,
   - with more than one monitor, Johnny on every monitor (each with a story of its own; only
     the main one is heard) or on the main monitor only, the others black,
   - the folder with the game files, if they are somewhere else. It tells you whether the files
     are found.

Any key, mouse click or moving the mouse ends the screensaver. Johnny's story moves on a day each
day the screensaver runs; it is kept with the settings in `%APPDATA%\JohnnyCastawayHD`.

The screensaver isn't signed, so Windows may warn about it the first time (SmartScreen: choose
**More info**, then **Run anyway**).

## Building

Requires [Go](https://go.dev/dl/) 1.27 or newer. No C compiler is needed.

```
go build -o bin/jchd.exe ./cmd/jchd
```

The same program built as `JohnnyCastawayHD.scr` is the screensaver:

```
go build -trimpath -ldflags=-H=windowsgui -o bin/JohnnyCastawayHD.scr ./cmd/jchd
```

`-H=windowsgui` keeps Windows from opening a console window when it starts the screensaver; the
price is that its log goes nowhere, so use `jchd.exe` to see it. `-trimpath` leaves the folders it
was built in out of the program.

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
