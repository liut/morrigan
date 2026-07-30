package convo

import (
	"math"
	"time"

	oid "github.com/cupogo/andvari/models/oid"
)

// MemoryDecayMeta holds the decay-related fields loaded for re-ranking.
type MemoryDecayMeta struct {
	ID             oid.OID
	Tier           string
	DecayRate      float64
	LastAccessedAt time.Time
}

// CalcRetention calculates the Ebbinghaus retention factor R = e^(-t / (24 * S))
// where t is hours since last access and S is the decay rate.
// Returns 1.0 when lastAccessedAt is zero (new memory) or decayRate <= 0.
func CalcRetention(lastAccessedAt time.Time, decayRate float64) float64 {
	if lastAccessedAt.IsZero() {
		return 1.0
	}
	if decayRate <= 0 {
		decayRate = 1.0
	}
	t := time.Since(lastAccessedAt).Hours()
	return math.Exp(-t / (24.0 * decayRate))
}

// CalcReinforce computes reinforced retention: R_new = min(1.0, R + factor * (1.0 - R)).
func CalcReinforce(currentR, factor float64) float64 {
	if currentR >= 1.0 {
		return 1.0
	}
	r := currentR + factor*(1.0-currentR)
	if r > 1.0 {
		return 1.0
	}
	return r
}

// TierDecayRate returns the decay rate for a given tier.
// Default ratios: working=1, short-term=7, long-term=60.
func TierDecayRate(tier string) float64 {
	switch tier {
	case "long-term":
		return 60.0
	case "short-term":
		return 7.0
	default:
		return 1.0
	}
}

// ForgottenMultiplier returns 0.1 when retention falls below forgetThreshold,
// otherwise 1.0. This dramatically down-ranks forgotten memories.
func ForgottenMultiplier(retention, forgetThreshold float64) float64 {
	if retention < forgetThreshold {
		return 0.1
	}
	return 1.0
}

// RouteTier assigns a tier based on importance score and configurable thresholds.
func RouteTier(score, longTermThreshold, shortTermThreshold float64) string {
	if score >= longTermThreshold {
		return "long-term"
	}
	if score >= shortTermThreshold {
		return "short-term"
	}
	return "working"
}

// PromoteTier checks whether a memory should be promoted based on access count.
// Returns the new tier if promotion occurred, empty string otherwise.
func PromoteTier(currentTier string, accessCount, promoteW2S, promoteS2L int) string {
	if currentTier == "working" && accessCount >= promoteW2S {
		return "short-term"
	}
	if currentTier == "short-term" && accessCount >= promoteS2L {
		return "long-term"
	}
	return ""
}
