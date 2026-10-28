# Original game files

The engine expects these files in `gamedata/`.

| File | Size (bytes) | MD5 | Source of reference hash |
|---|---|---|---|
| `SCRANTIC.SCR` | 295,952 | `48b6e36cac54544834c431f6f49c7ada` | Our install floppy (no other published hash) |
| `RESOURCE.MAP` | 1,461 | `374e6d05c5e0acd88fb5af748948c899` | Johnny-Castaway-Enhanced "Tested Files" |
| `RESOURCE.001` | 1,175,645 | `8bb6c99e9129806b5089a39d24228a36` | Johnny-Castaway-Enhanced "Tested Files" |

Compute with PowerShell: `Get-FileHash gamedata\* -Algorithm MD5`

## From the install floppy

The 1993 install floppy (1.44 MB, FAT12, files dated 1993-01-04) holds:

| File | Size | Notes |
|---|---|---|
| `SETUP.EXE` | 47,616 | Installer (16-bit Windows) |
| `INSTALL.EX$` | 102,948 | Packed `INSTALL.EXE` |
| `INSTALL.INS` | 2,910 | Installer script: renames and copies below |
| `SCRANTIC.SC$` | 158,517 | Packed; header name `SCRANTIC.EXE`, installed as `SCRANTIC.SCR` |
| `RESOURCE.00$` | 1,072,707 | Packed; installed as `RESOURCE.001` |
| `RESOURCE.MAP` | 1,461 | Copied as is |
| `RESOURCE.001` | 35 | Placeholder text ("Resource #1 has been compressed."), overwritten on install |
| `LOGO.BMP`, `SLOGO.BMP` | 630, 7,318 | Installer graphics |

Copy the files off the floppy (or out of a disk image with 7-Zip) into `gamedata/floppy/`,
then run `go run ./cmd/jcunpack`. It decompresses the packed files, checks the hashes above
and writes the three game files to `gamedata/`. Packed format: [install-format.md](install-format.md).

`SCRANTIC.SCR` is a 16-bit Windows (NE) executable. Its resource table holds 23 resources
of type `#28677`, each a RIFF WAVE sound.
