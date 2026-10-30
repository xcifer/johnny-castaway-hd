# Credits

*Screen Antics: Johnny Castaway* (1992) was created by Sierra On-Line / Dynamix.

This engine learns from, and ports parts of, these projects:

| Project | License | Used for |
|---|---|---|
| [jno6809/jc_reborn](https://github.com/jno6809/jc_reborn) | GPL-3.0 | RLE/LZW decompression (ported in `internal/resource/compress.go`), TTM interpreter and thread model (`internal/script/ttm.go`; frame timing was measured from the original instead), ADS scene interpreter (`internal/script/ads.go`), line and circle drawing (ported in `internal/script/draw.go`), story table, scene picking and island setup (`internal/story/story.go`), walking routes and turns between spots (`internal/story/walk.go`), resource layouts, opcode names and argument counts, sound file hashes, which sound each sample number plays (`internal/audio/audio.go`; checked against recordings of the original), layer model, tick rate, island layout, walk table location |
| [xesf/castaway](https://github.com/xesf/castaway) | MIT | Resource file format documentation, palette and sound offset tables used as cross-checks |
| [ScummVM](https://www.scummvm.org/) `engines/dgds` | GPL-3.0 | Dynamix DGDS formats and decompression |
| [fbreve/Johnny-Castaway-Enhanced](https://github.com/fbreve/Johnny-Castaway-Enhanced) | none stated | Behavioral reference only. **No code copied.** |

Libraries:

| Library | License |
|---|---|
| [Ebitengine](https://ebitengine.org/) | Apache-2.0 |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | BSD-3-Clause |
| `blast.c` by Mark Adler, from [zlib](https://zlib.net/) `contrib/blast` (ported to Go in `internal/install/explode.go`) | zlib |

Where code is ported from a reference project, the source file names the original file it came from.

The release download includes THIRD-PARTY-NOTICES.txt with the full license texts of everything built into the programs, including the libraries Ebitengine uses and the Go standard library.
