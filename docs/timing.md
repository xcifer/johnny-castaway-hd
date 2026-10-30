# Timing

## The clock

Script delays count in **ticks of 1/64 s (15.625 ms)**, and the screen is **redrawn every 4
ticks (62.5 ms, 16 times a second)**. A frame stays on screen for its delay rounded up to whole
redraws, and for at least one redraw:

| Delay asked for (ticks) | On screen |
|---|---|
| 0–4 | 62.5 ms |
| 5–8 | 125 ms |
| 9–12 | 187.5 ms |
| 13–16 | 250 ms |
| *n* | ⌈*n* / 4⌉ × 62.5 ms, at least 62.5 ms |

`TIMER a b` waits a random number of ticks from *a* to *b*. There is no minimum delay: a delay
of 0 shows the frame for one redraw.

Both clocks are in `internal/script` (`TicksPerSecond`, `FrameTicks`). `cmd/jchd` runs
Ebitengine at 64 ticks a second and moves the scene on every fourth tick. Drawing happens at the
display's refresh rate and always shows the latest state.

| Other timing | Value | Source |
|---|---|---|
| Johnny walking between spots | one step per redraw | measured |
| Pause after arriving | about 13 redraws (0.8 s) | measured, includes starting the next scene |
| Shore waves | every 2 redraws | jc_reborn's 8 ticks, not yet checked |
| Title picture at the start | 10 redraws (0.625 s) | measured: 0.584, 0.634 and 0.600 s |
| Black screen after the title and after each run | 12 redraws (0.75 s) | measured: 0.834, 0.801, 0.751 s after the title; 1.218 and 0.751 s between runs. The shortest is taken; the longer ones are presumably the original loading. |

## How it was measured

Two recordings of the original `SCRANTIC.SCR` running under winevdm on Windows 11 (1920×1080,
59.94 fps, game window at 75%, 60 s and 6.5 min) were analyzed frame by frame:

1. Every captured frame was compared with the one before it; each change and where it happened
   was logged.
2. Clips were identified by drawing them with this engine and matching every captured frame
   against the drawn frames (the island offset was found by the same matching). That gives the
   time each clip frame appeared in the original.

Findings:

- **Every change in both recordings falls on a 62.5 ms grid.** Gaps between changes cluster at
  62.5, 125, 187.5, 250 and 312.5 ms, with almost nothing in between.
- Frames with the same delay take the same time, and the times fit the rule above:

| Clip | Delay | Rule | Measured |
|---|---|---|---|
| MJSAND.TTM plane circling (10-frame loop, 6 loops) | 3 | 62.5 ms | 62.85 ms |
| MJFISHC.TTM tag 17, walk with the pole (2 recordings) | 7 | 125 ms | 124.6, 125.1 ms |
| MJFISHC.TTM tag 63, carrying the fish (18 frames) | 7 | 125 ms | 128.8 ms |
| MJFISHC.TTM tag 2, casting (2 recordings) | 8 | 125 ms | 140, 143 ms |
| MJFISHC.TTM tag 6, reeling in | 10 | 187.5 ms | 183, 184, 200 ms |
| MJFISHC.TTM tag 8, slow reeling (3 recordings) | 16 | 250 ms | 267, 269, 277 ms |

  Delays that are a multiple of 4 (8, 16) sometimes take one redraw more in the recordings:
  their time lands exactly on a redraw, and the Windows timer jitters around it. The engine
  takes the shorter time.

- **`TIMER` is random.** The same `TIMER 120 240` (holding the cast line) waited 2.67, 2.97 and
  3.37 s in three casts, all inside 120–240 ticks (1.9–3.75 s). `TIMER 60 120` waited 1.49 s
  (60–120 ticks: 0.94–1.88 s).
- **No 4-tick minimum.** jc_reborn raises delays below 4 to 4 ticks of 20 ms (80 ms); the plane
  loop at delay 3 runs at 62.5 ms per frame. No run of one-frame (17 ms) changes appears
  anywhere, which fits a delay of 0 lasting one redraw.
- **Walking.** Steps of the walk table (about 6.5 pixels each) come every 62.5 ms, sometimes
  125 ms; jc_reborn uses 6 ticks of 20 ms (120 ms).

The rule matches the 1/64 s clock of modern Windows: the original most likely asks for a timer
of about 55 ms, which Windows rounds up to 62.5 ms. On Windows 3.1 the timer ran in 55 ms steps,
so the original then redrew about 18 times a second; that can't be measured without period
hardware. This engine follows the recordings.

Earlier versions used jc_reborn's clock (50 ticks a second, delays of at least 4); scripts now
run about 15% faster, as in the recordings.

## Measured on this engine

`jchd -frames N` logs the measured rate on exit (time from the first tick to the last).

| Run | Ticks | Time | Rate |
|---|---|---|---|
| Windowed, 1280×720 | 640 | 9.92 s | 64.43/s |
| Windowed, 1280×720, `-ttm MJFISHC:43,17` | 1920 | 29.91 s | 64.17/s |
| Fullscreen | 640 | 9.92 s | 64.40/s |

Runs finish a little early: Ebitengine runs a few ticks back to back right after startup. The
steady rate is the difference between the windowed runs: 1280 ticks in 19.99 s = 64.03/s.

Rendering `-ttm MJFISHC:43,17` one tick at a time with `-shot` shows the picture changing every
8 ticks (125 ms) and never in between.
