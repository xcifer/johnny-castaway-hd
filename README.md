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

Run `bin/jchd.exe` (fullscreen; any key or mouse click exits). It currently shows the island
with Johnny walking.

| Option | Effect |
|---|---|
| `-windowed` | Run in a 1280×720 window |
| `-debug` | Keys control the display: F1 info overlay, F2 scaling filter, Esc exits |
| `-filter sharp\|nearest\|smooth` | How the original art is scaled up (default `sharp`) |
| `-data DIR` | Game data folder (default `gamedata`) |
| `-seed N` | Fixed random seed, for repeatable runs |
| `-shot FILE.png -shotsize 3840x2160 -shotticks N` | Render one frame at any size after N ticks, save it and exit |

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
