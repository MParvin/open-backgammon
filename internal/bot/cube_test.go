package bot

import (
	"testing"

	"bg/internal/game"
)

func TestCubeOfferBlockedOnCrawford(t *testing.T) {
	b := game.NewBoard()
	m := game.NewMatch(5)
	m.Crawford = true
	c := game.NewCube()
	if ShouldOffer(b, game.White, c, m, HardProfile()) {
		t.Fatal("no double in crawford")
	}
}

func TestOpeningDoesNotDouble(t *testing.T) {
	b := game.NewBoard()
	m := game.NewMatch(5)
	if ShouldOffer(b, game.White, game.NewCube(), m, HardProfile()) {
		t.Fatal("opening should not be an automatic double")
	}
}

func TestModerateFavoriteOffers(t *testing.T) {
	var b game.Board
	b.Off[0] = 8
	b.Points[1] = 4
	b.Points[2] = 3
	b.Off[1] = 2 // opponent has borne off: no gammon
	b.Points[24] = -6
	b.Points[23] = -7
	m := game.NewMatch(7)
	c := game.NewCube()
	if !ShouldOffer(b, game.White, c, m, HardProfile()) {
		t.Fatalf("moderate favorite should offer, eq=%v", PointEquity(b, game.White, 1))
	}
}

func TestTooGoodToDouble(t *testing.T) {
	var b game.Board
	b.Off[0] = 14
	b.Points[1] = 1
	b.Points[24] = -15
	m := game.NewMatch(11)
	if ShouldOffer(b, game.White, game.NewCube(), m, HardProfile()) {
		t.Fatal("gammon position should not be doubled away")
	}
	if ShouldTake(b, game.Black, game.NewCube(), m, HardProfile()) {
		t.Fatal("trailer should drop a lost position")
	}
}

func TestOwnedCubeBlocksOffer(t *testing.T) {
	c := game.NewCube().Take(game.Black)
	var b game.Board
	b.Off[0] = 8
	b.Points[1] = 7
	b.Off[1] = 1
	b.Points[13] = -14
	if ShouldOffer(b, game.White, c, game.NewMatch(5), HardProfile()) {
		t.Fatal("white does not own the cube")
	}
}
