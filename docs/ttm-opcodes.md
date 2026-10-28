# TTM opcodes

TTM scripts hold the animations. A script is a set of numbered clips: each starts at a `TAG`
and normally ends at a `PURGE`. The scene scripts (ADS) start clips with `ADD_SCENE slot tag
count weight`, often several at once: one clip loads the pictures and ends straight away,
another draws with them. Bitmap slots belong to the script, so all its clips share them.

The interpreter is `internal/script`. Its behavior follows jc_reborn (`ttm.c`, `graphics.c`, the
thread loop in `ads.c`). Where jc_reborn guesses, so do we, and it says so below.

## Threads and timing

- Each playing clip is a thread with its own transparent layer, in one of 10 slots. A new
  thread takes the lowest free slot, and higher slots are drawn on top. A new thread starts
  with an empty layer, no clip zone, bitmap slot 0, colors 15/15 and a delay of 4 ticks.
- A thread runs its code up to `UPDATE`; that frame then stays on screen for the thread's delay.
- `GOTO_TAG` takes effect after the current frame, not at once.
- When a clip ends (at `PURGE` or the end of the code), its last frame stays up for one more
  delay, then the thread stops and its layer disappears.
- `ADD_SCENE` count: *n* > 0 plays the clip *n* times; *n* < 0 loops it for −*n* ticks, with
  `PURGE` jumping back to the clip's start; 0 plays it once. A clip that is already playing
  isn't started again.

The tests check each of these rules (`internal/script/ttm_test.go`).

## Opcodes

All 30 opcodes used in the data are known. Uses: how often each occurs in the 41 scripts.

Status: **done**: implemented with a visible effect. **ignored**: read and skipped; no visible
effect is known. **guess**: implemented, but how the original behaves is uncertain.

| Opcode | Name | Uses | Status | What it does here |
|---|---|---:|---|---|
| 0x0080 | DRAW_BACKGROUND | 190 | ignored | Appears where clips free their pictures (e.g. GFFFOOD.TTM tag 11). |
| 0x0110 | PURGE | 528 | done | Ends the clip; in a timed loop, jumps back to the clip's start. |
| 0x0FF0 | UPDATE | 6631 | done | Ends the frame. |
| 0x1021 | SET_DELAY *n* | 1148 | guess | Frame time *n* ticks, at least 4. The minimum comes from jc_reborn; 93 uses ask for 0. |
| 0x1051 | SET_BMP_SLOT *n* | 456 | done | Selects bitmap slot *n* (0–5) for `LOAD_IMAGE`. |
| 0x1061 | SET_PALETTE_SLOT | 41 | ignored | The game has one palette. |
| 0x1101 | LOCAL_TAG *n* | 5 | done | A jump target, like `TAG`. |
| 0x1111 | TAG *n* | 611 | done | Start of clip *n*. Running into the next tag just continues. |
| 0x1121 | (unknown) | 61 | ignored | Comes before `SAVE_IMAGE1`; jc_reborn suspects it numbers a region for `CLEAR_SCREEN` (see WOULDBE.TTM). |
| 0x1201 | GOTO_TAG *n* | 75 | done | Continues at tag *n* after this frame. |
| 0x2002 | SET_COLORS *fg bg* | 217 | done | Colors for lines, pixels, rectangles and circles. Color 0 is transparent and erases. SASKDATE.TTM uses 207 once; we use its low 4 bits (15), a guess. |
| 0x2012 | SET_FRAME1 | 52 | ignored | Always `0 0`, next to `LOAD_IMAGE`. |
| 0x2022 | TIMER *a b* | 152 | guess | Frame time (*a*+*b*)/2 ticks. jc_reborn's formula, which it calls unsure. |
| 0x4004 | SET_CLIP_ZONE *x1 y1 x2 y2* | 52 | done | Limits drawing on the layer (*x2*, *y2* exclusive). In widescreen, a zone touching the edge of the 640×480 area reaches the edge of the frame. |
| 0x4204 | COPY_ZONE_TO_BG *x y w h* | 38 | done | Copies the layer's pixels into the saved zones, which stay when the layer is cleared. Copies *w*+2 columns, as jc_reborn does: GJVIS6.TTM's ship hull has a 2-pixel gap otherwise. |
| 0x4214 | SAVE_IMAGE1 | 60 | ignored | jc_reborn ignores it too. |
| 0xA002 | DRAW_PIXEL *x y* | 63 | done | One pixel in the foreground color. |
| 0xA054 | SAVE_ZONE | 1 | ignored | Only in GJGULIVR.TTM. |
| 0xA064 | RESTORE_ZONE | 1 | done | Clears all saved zones. Only in GJGULIVR.TTM, where no other zone is saved. |
| 0xA0A4 | DRAW_LINE *x1 y1 x2 y2* | 784 | done | Bresenham line in the foreground color, without its end point. |
| 0xA104 | DRAW_RECT *x y w h* | 111 | done | Filled rectangle in the foreground color. |
| 0xA404 | DRAW_CIRCLE *x y w h* | 437 | done | Circle filling the box: inside in the background color, outline in the foreground color. All circles in the data are round with an even diameter. |
| 0xA504 | DRAW_SPRITE *x y n slot* | 12546 | done | Sprite *n* of bitmap *slot*. |
| 0xA524 | DRAW_SPRITE_FLIP *x y n slot* | 2822 | done | The same, mirrored left to right within its box. |
| 0xA601 | CLEAR_SCREEN *n* | 6814 | done | Clears the whole layer, ignoring the clip zone. The argument (0, 1 or 2) is ignored. |
| 0xB606 | DRAW_SCREEN *x y w h* 2 1 | 6 | ignored | Ignored as in jc_reborn, but its arguments are a rectangle, so it may well draw something. To check against recordings. |
| 0xC051 | PLAY_SAMPLE *n* | 535 | done | Plays sound *n* (no sound output yet). |
| 0xF01F | LOAD_SCREEN *name* | 40 | done | Replaces the background and clears the saved zones. |
| 0xF02F | LOAD_IMAGE *name* | 263 | done | Loads a BMP into the selected bitmap slot. |
| 0xF05F | LOAD_PALETTE | 41 | ignored | Always JOHNCAST.PAL. |

An opcode not in this table is reported once and skipped.

Coordinates are signed. Pixels outside the 640×480 area are dropped in the original; in
widescreen they can show in the strips beside it.

## Checks

- `internal/script/testdata/scripts.md5` fingerprints every script played start to finish,
  sampled every tenth frame. It holds hashes only, no game data.
- Every clip that a scene script starts or stops exists in its script.
- The scaled renderer was compared with the 640×480 reference canvas at 3× with nearest
  scaling, on 40 frames of 8 scripts covering every drawing opcode. Every pixel inside the
  640×480 area matched, apart from column 639 where the background repair applies.

## Watching a script

```
jchd -windowed -debug -ttm MJFISH:1,18
```

plays clips 1 and 18 of MJFISH.TTM together over the island, then again after a second.
Include the clip that loads the pictures (usually 1), as the scene scripts do. `-ttm MJFISH`
plays the whole script in file order, which can draw a clip before the pictures it expects are
loaded. F1 shows which clips are playing.
