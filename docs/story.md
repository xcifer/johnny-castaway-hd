# The story

`internal/story` decides what plays. It follows `story.c`, `story_data.h`, `island.c`,
`walk.c` and `calcpath.c` of [jc_reborn](https://github.com/jno6809/jc_reborn), checked against
recordings of the original where they show it.

## Runs

The screensaver plays **runs**, one after another. A run:

1. Picks a final scene at random among the scenes that can end a run (and suit the day of the
   story).
2. Sets up the island for it: where it is, low tide, how much of the raft is built, night,
   holiday decorations.
3. Unless the final scene must also be the first, plays scenes that suit that island first.
   After each scene there is one more with chance (19 − *k*)/14, where *k* is the number played
   so far: always at least 6 and at most 19, most often 8 to 10.
4. Plays the final scene.

Between two scenes Johnny walks from where the last one left him to where the next one starts.

The table of 63 scenes is `story.Scenes`. Each has a start and end spot (A to F) and heading,
an optional day of the story, and flags:

| Flag | Meaning |
|---|---|
| `Final` | Can end a run |
| `First` | Can only be the first scene of a run |
| `OnIsland` | Plays on the island (the others bring their own pictures) |
| `LeftIsland` | Drawn for the island 272 pixels to the left |
| `AnyPlace` | Works wherever the island is |
| `LowTideOK` | Works at low tide |
| `NoRaft` | The raft isn't there |
| `NoHoliday` | Holiday decorations would be drawn over the scene (the cargo ship) |

Of the 66 scenes in the data, the story leaves out BUILDING.ADS 8 and 9 (the same as 7 and 5)
and STAND.ADS 14.

## The island

| What | Rule | Checked against the original |
|---|---|---|
| Position | For `AnyPlace` final scenes, one of three regions at random; else 272 px left for `LeftIsland`, or the usual place | Three offsets measured: (1, 20), (−77, 85), (−61, 74). The second is 1 px below the region (y −14 to 84) |
| Low tide | Half the time, if the final scene allows it | — |
| Raft | Stage 1 on days 1–2, then one more each day to stage 5 on day 6 | Stage 1 and its place, measured |
| Clouds | 0 to 5, most often 1 or none, all blown the same way | Clouds at night: seen |
| Night | Hour mod 8 is 0 or 7: 23–1, 7–9 and 15–17 o'clock | Day at 6:22 and 12:23, night at 16:11 |
| Holidays | By date, see below | Halloween pumpkin and its place, measured |

| Holiday | Dates | Decoration |
|---|---|---|
| Halloween | 29–31 October | Pumpkin |
| St Patrick's Day | 15–17 March | Shamrocks |
| Christmas | 23–25 December | Tree |
| New Year | 29 December – 1 January | Banner on the palm |

Decorations are drawn on top of everything, Johnny included.

## The day of the story

The story has 11 days; some scenes play only on one of them. The day moves on once for each new
date the screensaver runs (not for each day of the calendar), and after day 11 it starts again
at 1. It is kept in `%APPDATA%\JohnnyCastawayHD\progress.json`. `-day` and `-date` change it for
testing without saving.

## Walking

Spots A to F are places on the island. Johnny can only go from one spot to some others, and
which ones can depend on the spot he came from; a route never passes a spot twice, and one is
picked at random among all possible routes. At each spot he turns on the spot the shorter way,
then walks the stretch to the next one from the walk table in `SCRANTIC.SCR`. Arrived, he turns
to face the way the next scene starts, with his hands in his pockets. Walking between D and E
he passes behind the palm tree.

Each picture of a walk shows for one redraw, the last for about 13 (see [timing.md](timing.md)).

## Not yet done

Fading out between runs, the sounds, and the shore waves' timing, which comes from jc_reborn and
hasn't been checked against the original.
