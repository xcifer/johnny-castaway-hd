# ADS opcodes

ADS scripts hold the scenes. Each of the 10 scripts loads up to five TTM scripts into numbered
slots and holds several scenes, each starting at a tag. A scene's code starts clips (`ADD_SCENE
slot tag count weight`), and blocks headed `IF_LASTPLAYED slot tag` start more when a clip ends.
A scene is over when no clip plays.

The interpreter is `internal/script/ads.go`. It follows jc_reborn (`ads.c`).

## How a scene runs

- The scene's first block runs from its tag. A block ends at `PLAY_SCENE`, or at an
  `IF_LASTPLAYED` that doesn't follow `OR`.
- The scene waits for the clips named by the `IF_LASTPLAYED`s between its tag and the next tag,
  and by the `IF_NOT_RUNNING`s before the first `IF_LASTPLAYED` or `IF_IS_RUNNING`. When such a
  clip ends for good (not between repeats), its blocks run, in the same redraw.
  - In STAND.ADS the leading `IF_NOT_RUNNING`s keep Johnny idling: whenever one idle clip ends,
    another is picked.
- `IF_NOT_RUNNING` and `IF_IS_RUNNING` skip the rest of the block up to the next `PLAY_SCENE` or
  `END`, which then only ends the skip. A clip in its last frame counts as not running.
- After an `END` that isn't skipped, ending clips start nothing more and the scene runs out.

The tests check each of these rules (`internal/script/ads_test.go`).

## Opcodes

All 18 opcodes used in the data are known. Uses: how often each occurs in the 10 scripts.
Values that aren't opcodes are tag labels.

| Opcode | Name | Uses | Status | What it does here |
|---|---|---:|---|---|
| 0x1070 | IF_LASTPLAYED_LOCAL *slot tag* | 1 | done | Only in ACTIVITY.ADS tag 7: waits for the clip in place of the scene's own blocks, once. |
| 0x1330 | IF_UNKNOWN_1 *slot tag* | 55 | ignored | Comes before the `ADD_SCENE` of a clip that loads pictures; jc_reborn ignores it too. |
| 0x1350 | IF_LASTPLAYED *slot tag* | 500 | done | Heads a block that runs when the clip ends. |
| 0x1360 | IF_NOT_RUNNING *slot tag* | 76 | done | Skips if the clip plays. |
| 0x1370 | IF_IS_RUNNING *slot tag* | 30 | done | Skips if the clip doesn't play. |
| 0x1420 | AND | 47 | done | Between conditions; nothing to do, as each condition can only start a skip. |
| 0x1430 | OR | 30 | done | The next `IF_LASTPLAYED` doesn't end the block. |
| 0x1510 | PLAY_SCENE | 584 | done | Ends the block, or a skip. |
| 0x1520 | ADD_SCENE_LOCAL | 1 | done | Only after `IF_LASTPLAYED_LOCAL`: its five words are 0x2005 and the arguments of an `ADD_SCENE`, which runs when the local wait is over. |
| 0x2005 | ADD_SCENE *slot tag count weight* | 866 | done | Starts a clip; count as in [ttm-opcodes.md](ttm-opcodes.md). The weight counts only in a random block. |
| 0x2010 | STOP_SCENE *slot tag weight* | 111 | done | Stops the clip at once, without running its blocks. |
| 0x3010 | RANDOM_START | 77 | done | Starts a random block: its `ADD_SCENE`, `STOP_SCENE` and `NOP` are options. |
| 0x3020 | NOP *weight* | 17 | done | The option to do nothing. |
| 0x30FF | RANDOM_END | 77 | done | Carries out one option, picked by weight. |
| 0x4000 | UNKNOWN_6 *a b c* | 4 | ignored | Only in BUILDING.ADS; no visible effect known. |
| 0xF010 | FADE_OUT | 67 | ignored | Before the `END` of a scene. Probably fades the screen out; to be done with the story. |
| 0xF200 | GOSUB_TAG *tag* | 12 | done | Runs the block of another tag (STAND.ADS: tag 14 starts the standing clip). |
| 0xFFFF | END | 133 | done | Nothing more starts once running clips end (or ends a skip). |

## Mistakes in the original scripts

Playing every scene shows a few drawing commands that can't work in the original either. They
are reported once and skipped, and the tests expect exactly these:

| Script | Offset | Problem | Effect |
|---|---|---|---|
| MJDIVE.TTM | 66 | Draws from a bitmap slot before loading it | None: the same sprite is drawn again after loading, before the frame is shown |
| GJVIS5.TTM | 3478 | Bitmap slot 1 is never loaded | None: the sprite would be off screen |
| GJGULIVR.TTM | 3614 | Draws from LILIPUTS.BMP one clip before it is loaded | One sprite missing in the Lilliputians' boat scene |
| GJNAT1.TTM | 518, 6556 | Sprites past the end of the bitmap | Nothing drawn |
| GJVIS3.TTM | 4356, 4496 | Sprite past the end of the bitmap | Nothing drawn |
| SMDATE.TTM | 6332, 6362, 6388 | Sprites past the end of the bitmap | Nothing drawn |
| SASKDATE.TTM | 1928 | Sprite past the end of the bitmap | Nothing drawn |

## Checks

- Every one of the 66 scenes ends, played with 20 different random seeds. Most take 10 to 100
  seconds; the longest seen was 2½ minutes (ACTIVITY.ADS tag 1).
- `internal/script/testdata/scenes.md5` fingerprints which clips each scene starts and when,
  for five seeds. It holds hashes only, no game data.
- FISHING.ADS scene 8 tells the same story as the recording of the original: walk to the shore
  with the rod, cast, wait, reel in, catch, carry the fish to the tree. The clips' frames were
  matched against the recording ([timing.md](timing.md)).

## Watching a scene

```
jchd -windowed -debug -left -ads FISHING:8
```

plays fishing scene 8 over the island, then again after a second. An unknown scene number lists
the script's scenes. Scenes that need the island elsewhere (here `-left`) can't tell yet; that
comes with the story, which also picks the scenes, walks Johnny between them and sets the time
of day.
