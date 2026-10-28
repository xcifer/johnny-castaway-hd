# Credits

*Screen Antics: Johnny Castaway* (1992) was created by Sierra On-Line / Dynamix.

This engine learns from, and ports parts of, these projects:

| Project | License | Used for |
|---|---|---|
| [jno6809/jc_reborn](https://github.com/jno6809/jc_reborn) | GPL-3.0 | RLE/LZW decompression (ported in `internal/resource/compress.go`), TTM interpreter and thread timing (`internal/script/ttm.go`), line and circle drawing (ported in `internal/script/draw.go`), resource layouts, opcode names and argument counts, sound file hashes, layer model, tick rate, island layout, walk table location |
| [xesf/castaway](https://github.com/xesf/castaway) | MIT | Resource file format documentation, palette and sound offset tables used as cross-checks |
| [ScummVM](https://www.scummvm.org/) `engines/dgds` | GPL-3.0 | Dynamix DGDS formats and decompression |
| [fbreve/Johnny-Castaway-Enhanced](https://github.com/fbreve/Johnny-Castaway-Enhanced) | none stated | Behavioral reference only. **No code copied.** |

Libraries:

| Library | License |
|---|---|
| [Ebitengine](https://ebitengine.org/) | Apache-2.0 |
| `blast.c` by Mark Adler, from [zlib](https://zlib.net/) `contrib/blast` (ported to Go in `internal/install/explode.go`) | zlib |

Where code is ported from a reference project, the source file names the original file it came from.
