# Timing

## Tick rate

The game clock runs at **50 ticks per second (one tick = 20 ms)**. This follows jc_reborn
(`events.c`: `eventsWaitTick` multiplies delays by 20 ms). Script delays are counted in ticks:

| Source | Value |
|---|---|
| TTM `SET_DELAY n` | wait *n* ticks between frames; jc_reborn raises values below 4 to 4 (80 ms) |
| TTM `TIMER a b` | jc_reborn uses (a + b) / 2 ticks |
| Shore waves (island background) | every 8 ticks |
| Johnny walking | one step every 6 ticks; 80 ticks after arriving |

To be confirmed against recordings of the original once they exist.

## Implementation

`cmd/jchd` sets Ebitengine's update rate to 50 per second, and each `Update` call is one game
tick. Ebitengine keeps the average rate and catches up after a stall, so game time doesn't drift
with the display's refresh rate. Drawing happens at the display's refresh rate and always shows the
latest game state.

## Measured

`jchd -frames N` logs the measured rate on exit (time from the first tick to the last).

| Run | Ticks | Time | Rate |
|---|---|---|---|
| Windowed, 1280×720 | 500 | 9.89 s | 50.47/s |
| Fullscreen | 500 | 9.90 s | 50.39/s |
| Windowed, 1280×720 | 1500 | 29.88 s | 50.16/s |

Both windowed runs finish about 0.1 s early: Ebitengine runs a few ticks back to back right after
startup. The steady rate is the difference between the runs: 1000 ticks in 19.99 s = 50.03/s.

## The original, measured

A 60-second recording of the original `SCRANTIC.SCR` running under winevdm on Windows 11
(recorded at 1920×1080, 59.94 fps) was analyzed frame by frame. A patch of empty sky was used as a
control and never changed, so every detected change is real animation.

| Observation | Measured | Meaning |
|---|---|---|
| Walking steps | gaps alternate 117 / 133 ms, average **125.0 ms** | 6 ticks × 20 ms = 120 ms, rounded up to the next step of the Windows timer (15.625 ms): 125 ms |
| Other gaps | ~187, 250, ~281 ms | 180, 240, 280 ms rounded up the same way |
| Shore waves | **not animated continuously**: nothing changed for 2.5 s at 6.4–8.9 s | jc_reborn animates them every 8 ticks; to be checked when the scene scripts are implemented |

The original asks for delays in 20 ms steps, as jc_reborn assumes; the system timer stretches each
wait slightly. On 1992 hardware the Windows timer ran in 55 ms steps, which would have made waits
longer still; that can't be measured without period hardware or an accurate emulator.
