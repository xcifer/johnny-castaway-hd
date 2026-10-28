# Install floppy packed files (`*.??$`)

Each packed file holds one compressed file. All integers are little-endian.

| Offset | Size | Value |
|---|---|---|
| 0x00 | 4 | Signature `65 5D 13 8C` |
| 0x04 | 4 | `08 01 03 00` in every file seen |
| 0x08 | 4 | `1` in every file seen (probably the file count) |
| 0x0C | 2 | `00 12` in every file seen |
| 0x0E | 4 | Equals the compressed stream length for `SCRANTIC.SC$` and `INSTALL.EX$`, but not for `RESOURCE.00$` (1,055,070 vs 1,072,665). Not used. |
| 0x12 | 4 | Zero |
| 0x16 | 4 | Probably DOS date, then DOS time, of the original file (`SCRANTIC.SC$` decodes to 1992-12-09 17:08:52) |
| 0x1A | 2 | Zero |
| 0x1C | 1 | Length *n* of the file name |
| 0x1D | *n* | File name, e.g. `SCRANTIC.EXE` |
| 0x1D+*n* | 1 | `00` |
| 0x1E+*n* | rest | PKWARE DCL implode stream |

The stream starts with two bytes: literal mode (0 = raw bytes, 1 = Huffman-coded) and
dictionary size (4, 5 or 6 for 1, 2 or 4 KB). All three files seen use `00 06`. Each stream
ends with the end code and leaves no trailing bytes.

The decoder is `internal/install`, a port of `blast.c` from zlib's `contrib/blast`.
