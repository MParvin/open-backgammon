// Package bot implements Easy, Intermediate, Hard, and LLM-backed opponents.
package bot

import (
	"math/rand/v2"

	"bg/internal/game"
)

// Player chooses one complete legal play for a roll.
type Player interface {
	Name() string
	Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error)
}

// Easy uses EasyProfile.
type Easy struct {
	Rand *rand.Rand
}

func (Easy) Name() string { return EasyProfile().Name }

func (e Easy) Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error) {
	return Level{Profile: EasyProfile(), Rand: e.Rand}.Choose(b, c, d1, d2)
}

func (e Easy) Profile() Profile { return EasyProfile() }

// Intermediate uses IntermediateProfile.
type Intermediate struct {
	Rand *rand.Rand
}

func (Intermediate) Name() string { return IntermediateProfile().Name }

func (i Intermediate) Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error) {
	return Level{Profile: IntermediateProfile(), Rand: i.Rand}.Choose(b, c, d1, d2)
}

func (Intermediate) Profile() Profile { return IntermediateProfile() }

// Hard uses HardProfile.
type Hard struct {
	Rand *rand.Rand
}

func (Hard) Name() string { return HardProfile().Name }

func (h Hard) Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error) {
	return Level{Profile: HardProfile(), Rand: h.Rand}.Choose(b, c, d1, d2)
}

func (Hard) Profile() Profile { return HardProfile() }
