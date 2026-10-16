package bot

import "math/rand/v2"

// Profile configures how a difficulty level searches and selects moves.
// Profiles are centralized; callers should not sprinkle difficulty branches.
type Profile struct {
	Name string

	// Candidates is how many top static plays enter the search (or soft pick).
	Candidates int
	// Lookahead averages opponent replies over the 36 dice faces.
	Lookahead bool
	// NoiseStd is Gaussian noise added to each candidate's score before ranking.
	NoiseStd float64
	// SoftmaxTemp samples among candidates. Zero always picks the best score.
	SoftmaxTemp float64
	// BlunderProb is the chance to pick uniformly among the weaker half of candidates.
	BlunderProb float64

	// DoubleThreshold: offer a double when WinProb - TakeThreshold exceeds this.
	DoubleThreshold float64
	// TakeThreshold: take a double when WinProb is at least this (equity proxy).
	TakeThreshold float64
}

// EasyProfile is a light evaluator with intentional soft mistakes.
func EasyProfile() Profile {
	return Profile{
		Name:            "Easy",
		Candidates:      8,
		Lookahead:       false,
		NoiseStd:        22,
		SoftmaxTemp:     2.8,
		BlunderProb:     0.38,
		DoubleThreshold: 0.72,
		TakeThreshold:   0.28,
	}
}

// IntermediateProfile ranks more candidates with mild noise.
// Softmax keeps occasional soft errors without Easy-level blunders.
func IntermediateProfile() Profile {
	return Profile{
		Name:            "Intermediate",
		Candidates:      8,
		Lookahead:       false,
		NoiseStd:        5.5,
		SoftmaxTemp:     0.65,
		BlunderProb:     0.14,
		DoubleThreshold: 0.62,
		TakeThreshold:   0.32,
	}
}

// HardProfile uses lookahead and always takes the top score.
func HardProfile() Profile {
	return Profile{
		Name:            "Hard",
		Candidates:      12,
		Lookahead:       true,
		NoiseStd:        0,
		SoftmaxTemp:     0,
		BlunderProb:     0,
		DoubleThreshold: 0.58,
		TakeThreshold:   0.35,
	}
}

// Level is a Player driven by a Profile.
type Level struct {
	Profile Profile
	Rand    *rand.Rand
}

func (l Level) Name() string { return l.Profile.Name }

func NewEasy(r *rand.Rand) Level {
	return Level{Profile: EasyProfile(), Rand: r}
}

func NewIntermediate(r *rand.Rand) Level {
	return Level{Profile: IntermediateProfile(), Rand: r}
}

func NewHard(r *rand.Rand) Level {
	return Level{Profile: HardProfile(), Rand: r}
}
