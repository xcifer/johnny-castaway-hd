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

Run `bin/jchd.exe` (fullscreen; any key or mouse click exits) or `bin/jchd.exe -windowed`.

## License

GPL-3.0. See [LICENSE](LICENSE) and [CREDITS.md](CREDITS.md).

*Johnny Castaway* and its art, sounds and scripts are the property of their rights holders.
This is an unofficial fan project, not affiliated with or endorsed by Sierra, Dynamix,
Activision or Microsoft.
