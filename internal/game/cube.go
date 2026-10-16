package game

// MaxCube is the highest doubling-cube face supported.
const MaxCube = 32

// Cube is the doubling cube. Owner nil means the cube is centered.
type Cube struct {
	Value int
	Owner *Color
}

func NewCube() Cube {
	return Cube{Value: 1}
}

func (c Cube) Centered() bool {
	return c.Owner == nil
}

// CanOffer reports whether whose turn it is may offer a double.
// Doubling is illegal during a Crawford game, with a pending offer,
// after a finished game, or when the player does not own a non-centered cube.
func CanOffer(c Cube, turn Color, crawford bool, pending bool, finished bool) bool {
	if crawford || pending || finished {
		return false
	}
	if c.Value >= MaxCube {
		return false
	}
	if c.Owner != nil && *c.Owner != turn {
		return false
	}
	return true
}

// OfferedValue is the cube face after a take. The stored value is not raised until Take.
func (c Cube) OfferedValue() int {
	v := c.Value * 2
	if v > MaxCube {
		return MaxCube
	}
	return v
}

// Take accepts an offer: the cube rises and ownership passes to the taker.
func (c Cube) Take(taker Color) Cube {
	c.Value = c.OfferedValue()
	own := taker
	c.Owner = &own
	return c
}
