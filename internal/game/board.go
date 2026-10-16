// Package game is the backgammon rules engine: board, dice, cube, and match play.
package game

// Color is a player. White moves toward point 1 and bears off past 1.
// Black moves toward point 24 and bears off past 24.
type Color int8

const (
	White Color = 1
	Black Color = -1
)

func (c Color) Opponent() Color { return -c }

func (c Color) String() string {
	if c == White {
		return "White"
	}
	return "Black"
}

func colorIndex(c Color) int {
	if c == White {
		return 0
	}
	return 1
}

const (
	// Bar is the source of a checker waiting to re-enter.
	Bar = 0
	// Off is the destination of a borne-off checker.
	Off             = 25
	checkersPerSide = 15
)

// Board is a backgammon position. Points[1..24] store signed checker counts:
// positive for white, negative for black. Points[0] is unused.
type Board struct {
	Points [26]int8
	Bar    [2]int8
	Off    [2]int8
}

// NewBoard returns the standard opening position.
func NewBoard() Board {
	var b Board
	b.Points[24] = 2
	b.Points[13] = 5
	b.Points[8] = 3
	b.Points[6] = 5
	b.Points[1] = -2
	b.Points[12] = -5
	b.Points[17] = -3
	b.Points[19] = -5
	return b
}

func (b Board) Count(c Color, point int) int {
	n := int(b.Points[point])
	if c == White {
		if n > 0 {
			return n
		}
		return 0
	}
	if n < 0 {
		return -n
	}
	return 0
}

func (b Board) OnBar(c Color) int {
	return int(b.Bar[colorIndex(c)])
}

func (b Board) BorneOff(c Color) int {
	return int(b.Off[colorIndex(c)])
}

// Winner returns the color that has borne off all 15 checkers, or nil.
func (b Board) Winner() *Color {
	if b.BorneOff(White) >= checkersPerSide {
		c := White
		return &c
	}
	if b.BorneOff(Black) >= checkersPerSide {
		c := Black
		return &c
	}
	return nil
}

// Pips is the pip count for c (bar checkers count as 25).
func (b Board) Pips(c Color) int {
	n := b.OnBar(c) * 25
	for p := 1; p <= 24; p++ {
		cnt := b.Count(c, p)
		if c == White {
			n += cnt * p
		} else {
			n += cnt * (25 - p)
		}
	}
	return n
}

// AllInHome reports whether every checker of c is in that color's home board.
func (b Board) AllInHome(c Color) bool {
	if b.OnBar(c) > 0 {
		return false
	}
	if c == White {
		for p := 7; p <= 24; p++ {
			if b.Count(White, p) > 0 {
				return false
			}
		}
		return true
	}
	for p := 1; p <= 18; p++ {
		if b.Count(Black, p) > 0 {
			return false
		}
	}
	return true
}

func (b Board) remove(c Color, point int) Board {
	if c == White {
		b.Points[point]--
	} else {
		b.Points[point]++
	}
	return b
}

func (b Board) add(c Color, point int) Board {
	if c == White {
		b.Points[point]++
	} else {
		b.Points[point]--
	}
	return b
}
