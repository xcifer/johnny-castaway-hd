# Resource formats

How the original game data is stored, as implemented in `internal/resource`. All integers are
little-endian. Sources: jc_reborn (`resource.c`, `uncompress.c`, `ttm.c`, `ads.c`),
xesf/castaway (`docs/`, `src/dgds/`), and checks against the real files.

## Files

| File | Contents |
|---|---|
| `RESOURCE.MAP` | Index of the resource archive |
| `RESOURCE.001` | 180 resources: 117 BMP, 41 TTM, 10 ADS, 10 SCR, 1 PAL, 1 VIN |
| `SCRANTIC.SCR` | The Windows screensaver (16-bit NE executable); holds the 23 sounds |

## RESOURCE.MAP

| Size | Field |
|---|---|
| 6 | Header. Byte 4 is the number of resource files (always 1) |
| 13 | Resource file name, NUL-padded (`RESOURCE.001`) |
| 2 | Entry count *n* |
| 8 × *n* | Per entry: u32 (not used), u32 offset of the entry in the resource file |

## RESOURCE.001 entry

| Size | Field |
|---|---|
| 13 | Name, NUL-padded, e.g. `JOHNWALK.BMP` |
| 4 | Body size |
| … | Body: a series of blocks, each starting with a 4-character tag such as `BMP:` |

## Compressed blocks

`BIN:`, `VGA:`, `TT3:` and the `SCR:` block inside ADS files hold compressed data:

| Size | Field |
|---|---|
| 4 | Size of the rest of the block, including the next 5 bytes |
| 1 | Method: 0 none, 1 RLE, 2 LZW (the game uses 1 and 2) |
| 4 | Decompressed size |
| … | Data |

**RLE:** a control byte with bit 7 set repeats the next byte `control & 0x7F` times. Otherwise the
next `control` bytes are copied.

**LZW:** codes of 9 to 12 bits, least significant bit first. Code 256 clears the dictionary, skips
to the end of the current group of eight codes, and sets the next free entry to **256** (not 257).
The decoder reads one byte ahead. Ported exactly from jc_reborn.

Every compressed block in the game decodes to exactly its declared size and uses up its input exactly.

## Pictures

Pixels are 4-bit palette indices, two per byte, **high nibble first**.

**BMP (sprite sheet):** `BMP:` u16 u16 (unused) · `INF:` u32 size, u16 frame count, then all widths
(u16 each), then all heights (u16 each) · `BIN:` the frames' pixels one after another, as one
continuous stream of nibbles. `JOHNWALK.BMP` and `MJTELE.BMP` also have a `VGA:` block: the upper
4 bits of each pixel for 256-color mode, the same size as `BIN:`. Only six of its nibbles are
non-zero in each file. The 16-color game ignores it.

**SCR (background):** `SCR:` u16 u16 (unused) · `DIM:` u32 size, u16 width, u16 height · `BIN:` pixels.
Six backgrounds are 640×480; `ISLAND2.SCR`, `ISLETEMP.SCR`, `JOFFICE.SCR` and `THEEND.SCR` are
640×350. No sprite frame has an odd width.

**PAL:** `PAL:` u16 u8 u8 · `VGA:` 4 bytes · 256 RGB triplets, 6-bit (0–63). Only the first 16 colors
are used: the standard 16 Windows/VGA colors.

| Index | 6-bit RGB | Color |
|---|---|---|
| 0 | 42 0 42 | Magenta: **transparent** in sprites |
| 1 | 0 0 42 | Dark blue |
| 2 | 0 42 0 | Dark green |
| 3 | 0 42 42 | Dark cyan |
| 4 | 42 0 0 | Dark red |
| 5 | 0 0 0 | Black |
| 6 | 42 42 0 | Olive |
| 7 | 53 53 53 | Light gray |
| 8 | 32 32 32 | Gray |
| 9 | 0 0 63 | Blue |
| 10 | 0 63 0 | Green |
| 11 | 0 63 63 | Cyan |
| 12 | 63 0 0 | Red |
| 13 | 63 0 63 | Magenta |
| 14 | 63 63 0 | Yellow |
| 15 | 63 63 63 | White |

The art uses checkerboard dithering to fake in-between shades (red + white = Johnny's skin). Any
upscaling or filtering must take this into account.

## Scripts

**TTM (animation):** `VER:` u32 *n* + *n* bytes (`4.09`, one file `4.12`) · `PAG:` u32 pages + 2 bytes ·
`TT3:` compressed code · `TTI:` 4 bytes · `TAG:` u32 size, u16 count, then per tag u16 id + string.

TTM code is a list of u16 opcodes. The low nibble is the number of u16 arguments; `0xF` means one
NUL-terminated string argument, padded to an even length. `0x1111 n` marks the start of tag *n*.

**ADS (scene):** `VER:` as TTM (`1.09`) · `ADS:` 4 bytes · `RES:` u32 size, u16 count, then per entry
u16 slot + TTM name · `SCR:` compressed code · `TAG:` as TTM.

ADS code is a list of u16 values. Known opcodes have a fixed argument count
(`resource.ADSArgs`); any other value is a tag label.

Checks that pass on the real data:

- every script decodes to exactly its last byte;
- all 30 TTM opcodes in use are known (counts: `gamedata/dump/opcodes.txt`);
- every TTM `TAG` and every ADS label is in its file's tag list;
- `RANDOM_START` and `RANDOM_END` pair up (77 each);
- every file a script loads exists, except two (below).

Data quirks:

- `FIRE.TTM` loads `FLAME.BMP` and `FLURRY.BMP`, which aren't in the archive. No ADS uses
  `FIRE.TTM` (scenes use `MJFIRE.TTM`), so this never matters.
- `FIRE.TTM`, `GJVIS5W.TTM` and `MJCOCO1.TTM` are not used by any ADS.

## Sounds

`SCRANTIC.SCR` is a 16-bit Windows (NE) executable. Its resource table holds 23 resources of
type 28677 (`0x7005`), IDs 1–23. Each is a RIFF WAVE file, 8-bit mono PCM at 11,025 Hz, padded
to a multiple of 512 bytes. Resources 1–22 match the MD5s of the sound files listed by jc_reborn.

**Open question: which sound number plays which resource.** TTM `PLAY_SAMPLE n` uses
*n* = 1–12, 14–16, 18–24. The reference projects agree for 1–10 and that 11 has no sound, but
differ from 12 on:

| Sample | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 | 22 | 23 | 24 | 25 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| jc_reborn → resource | 11 | – | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 | 22 | – |
| xesf → resource | 12 | 11 | 15 | 22 | 13 | 14 | 23 | 16 | 17 | 18 | 19 | 20 | 21 | 22 |

To be settled by listening against the original before sound is wired up.
