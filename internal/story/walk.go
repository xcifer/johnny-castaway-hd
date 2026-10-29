package story

// Walking between spots follows walk.c, walk_data.h, calcpath.c and
// calcpath_data.h of jc_reborn by Jeremie Guillaume (GPL-3.0).

import (
	"math/rand/v2"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

const numSpots = 6

// The walk table (resource.WalkTable) holds a walk for some pairs of spots
// and, for each spot, Johnny turned to each of the 8 headings, first with
// his arms swinging and then, 9 steps further on, with his hands in his
// pockets. Indexes are steps of the table; -1 means no walk.
var (
	walkStart = [numSpots][numSpots]int{
		//A    B    C    D    E    F
		{-1, 68, 38, -1, 0, 17},       // from A
		{109, -1, 133, -1, -1, -1},    // B
		{163, 196, -1, 211, 224, 245}, // C
		{-1, -1, 278, -1, 289, 302},   // D
		{332, -1, 356, 381, -1, 394},  // E
		{423, -1, 443, 457, 463, -1},  // F
	}
	turnStart = [numSpots]int{91, 145, 260, 314, 405, 471}

	// Which way Johnny faces when he sets off from one spot to another, and
	// when he gets there.
	setOffFacing = [numSpots][numSpots]Heading{
		{-1, 6, 6, -1, 5, 5},
		{3, -1, 5, -1, -1, -1},
		{2, 1, -1, 3, 2, 2},
		{-1, -1, 7, -1, 2, 1},
		{1, -1, 7, 6, -1, 7},
		{2, -1, 6, 5, 3, -1},
	}
	arriveFacing = [numSpots][numSpots]Heading{
		{-1, 7, 5, -1, 5, 5},
		{3, -1, 5, -1, -1, -1},
		{2, 1, -1, 4, 3, 3},
		{-1, -1, 7, -1, 2, 1},
		{1, -1, 6, 6, -1, 7},
		{2, -1, 6, 5, 3, -1},
	}
)

// pocketsOffset is how much further in the table Johnny's turns with his
// hands in his pockets are.
const pocketsOffset = 9

// canWalk[from][at][to] tells whether Johnny, having come to spot at from
// spot from, can go on to spot to. from == numSpots is for the spot he
// starts at.
var canWalk = [numSpots + 1][numSpots][numSpots]bool{
	{ // came from A
		{},
		{SpotC: true},
		{SpotD: true},
		{},
		{SpotD: true, SpotF: true},
		{},
	},
	{ // B
		{SpotE: true},
		{},
		{SpotD: true},
		{},
		{},
		{},
	},
	{ // C
		{SpotE: true},
		{SpotA: true},
		{},
		{SpotE: true},
		{},
		{},
	},
	{ // D
		{},
		{},
		{SpotA: true, SpotB: true, SpotF: true},
		{},
		{SpotA: true},
		{SpotE: true},
	},
	{ // E
		{SpotB: true, SpotC: true},
		{},
		{},
		{SpotC: true},
		{},
		{SpotD: true},
	},
	{ // F
		{},
		{},
		{SpotD: true},
		{},
		{SpotA: true},
		{},
	},
	{ // starting out
		{SpotB: true, SpotC: true, SpotE: true, SpotF: true},
		{SpotA: true, SpotC: true},
		{SpotA: true, SpotB: true, SpotD: true, SpotE: true, SpotF: true},
		{SpotC: true, SpotE: true, SpotF: true},
		{SpotA: true, SpotC: true, SpotD: true, SpotF: true},
		{SpotA: true, SpotC: true, SpotD: true, SpotE: true},
	},
}

// Routes lists every way to walk from one spot to another, without passing
// a spot twice. Each route starts with from and ends with to.
func Routes(from, to Spot) [][]Spot {
	var routes [][]Spot
	seen := [numSpots]bool{}
	seen[from] = true
	route := []Spot{from}
	var walk func(prev int, at Spot)
	walk = func(prev int, at Spot) {
		if at == to {
			routes = append(routes, append([]Spot(nil), route...))
			return
		}
		for next := range Spot(numSpots) {
			if canWalk[prev][at][next] && !seen[next] {
				seen[next] = true
				route = append(route, next)
				walk(int(at), next)
				route = route[:len(route)-1]
				seen[next] = false
			}
		}
	}
	walk(numSpots, from)
	return routes
}

// WalkFrame is one picture of Johnny walking: a step of the walk table,
// and whether the palm tree hides him.
type WalkFrame struct {
	resource.WalkStep
	BehindTree bool
}

// Walk returns the pictures of Johnny walking from spot from, facing
// fromFacing, to spot to, facing toFacing, along a route picked at random.
// He turns on the spot before setting off and on arriving; the last
// picture shows him arrived, with his hands in his pockets.
func Walk(table []resource.WalkStep, rng *rand.Rand, from Spot, fromFacing Heading, to Spot, toFacing Heading) []WalkFrame {
	routes := Routes(from, to)
	return walkRoute(table, routes[rng.IntN(len(routes))], fromFacing, toFacing)
}

// walkRoute walks a route given by Routes.
func walkRoute(table []resource.WalkStep, route []Spot, fromFacing, toFacing Heading) []WalkFrame {
	from, to := route[0], route[len(route)-1]
	at, facing := from, fromFacing
	next, nextFacing := Spot(-1), toFacing // nextFacing -1: walking straight
	lastTurn, behindTree, arrived := true, false, false
	if at != to {
		route = route[1:]
		next, route = route[0], route[1:]
		nextFacing = setOffFacing[at][next]
		lastTurn = false
	}
	// The way to turn: +1 clockwise, -1 the other way, the shorter one.
	way := func() Heading {
		switch d := (nextFacing - facing) & 7; {
		case d == 0:
			return 0
		case d < 4:
			return 1
		}
		return -1
	}
	turn := way()

	var frames []WalkFrame
	i := 0
	for !arrived {
		switch {
		case nextFacing != -1 && ((nextFacing-facing)&7)%7 > 1:
			// Still turning.
			facing = (facing + turn) & 7
			i = turnStart[at] + int(facing)
			if lastTurn {
				i += pocketsOffset
			}
		case nextFacing != -1 && at != to:
			// Turned: set off for the next spot.
			nextFacing = -1
			behindTree = at == SpotD && next == SpotE || at == SpotE && next == SpotD
			i = walkStart[at][next]
		case nextFacing != -1:
			// Turned at the end of the walk.
			i = turnStart[to] + int(toFacing) + pocketsOffset
			arrived = true
		default:
			i++
			if table[i].X != 0 {
				break
			}
			// At the next spot: start turning to go on, or to stop.
			facing = arriveFacing[at][next]
			at = next
			if at != to {
				next, route = route[0], route[1:]
				nextFacing = setOffFacing[at][next]
			} else {
				nextFacing = toFacing
				lastTurn = true
			}
			turn = way()
			facing = (facing + turn) & 7
			i = turnStart[at] + int(facing)
			if lastTurn {
				i += pocketsOffset
				arrived = facing == toFacing
			}
		}
		frames = append(frames, WalkFrame{table[i], behindTree})
	}
	return frames
}
