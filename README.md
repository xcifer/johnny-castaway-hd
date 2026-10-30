# Johnny Castaway HD

A from-scratch engine for the 1992 Sierra/Dynamix screensaver *Screen Antics: Johnny Castaway*,
built as a native Windows 10/11 screensaver with optional HD art.

**Status:** early development. Nothing playable yet.

## You need your own copy of the original

This repository contains **no** original game files or art. The engine reads the files from a
copy of *Screen Antics: Johnny Castaway* that you own:

- `SCRANTIC.SCR`
- `RESOURCE.MAP`
- `RESOURCE.001`

For development, put them in a `gamedata/` folder in the repo root. Git ignores this folder.

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

Run `bin/jchd.exe` (fullscreen; any key or mouse click exits). It plays the story: scene after
scene on the island, with Johnny walking from one to the next. Night, holidays (Halloween
29-31 October, St Patrick's Day 15-17 March, Christmas 23-25 December, New Year 29 December
to 1 January) and the raft Johnny builds follow the clock and the day of the story, which
moves on once for each date the screensaver runs and is kept in
`%APPDATA%\JohnnyCastawayHD\progress.json`. See [docs/story.md](docs/story.md).

| Option | Effect |
|---|---|
| `-windowed` | Run in a 1280×720 window |
| `-debug` | Keys control the display: F1 info overlay, F2 scaling filter, Esc exits |
| `-filter sharp\|nearest\|smooth` | How the original art is scaled up (default `sharp`) |
| `-data DIR` | Game data folder (default `gamedata`) |
| `-seed N` | Fixed random seed, for repeatable runs |
| `-shot FILE.png -shotsize 3840x2160 -shotticks N` | Render one frame at any size after N ticks, save it and exit |
| `-ttm NAME` or `-ttm NAME:TAG,TAG` | Play one animation script, or some of its clips, over and over (see [docs/ttm-opcodes.md](docs/ttm-opcodes.md)) |
| `-ads NAME:SCENE` | Play one scene of a scene script over and over (see [docs/ads-opcodes.md](docs/ads-opcodes.md)) |
| `-left` | Move the island to the left, as some scenes do (for `-ttm` and `-ads`) |
| `-day N` | Play day N (1-11) of the story instead of the saved day, without saving |
| `-date "2026-12-24 15:30"` | Pretend the clock started at this time, for night and holidays, without saving |
| `-mute` | Play without sound |
| `-frames N` | Exit after N ticks (64 a second) and log the measured tick rate |

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
