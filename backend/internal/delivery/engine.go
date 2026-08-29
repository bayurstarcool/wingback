// Package delivery implements the core "carrier" mechanic: computing
// ETA between sender and recipient based on real GPS distance, carrier
// speed, and rolling the message-loss chance.
package delivery

import (
	"math"
	"math/rand"
	"time"
)

const earthRadiusKM = 6371.0

// Coordinates represents a WGS84 lat/lng pair.
type Coordinates struct {
	Lat float64
	Lng float64
}

// HaversineKM returns the great-circle distance in kilometers between two points.
func HaversineKM(a, b Coordinates) float64 {
	lat1 := degToRad(a.Lat)
	lat2 := degToRad(b.Lat)
	dLat := degToRad(b.Lat - a.Lat)
	dLng := degToRad(b.Lng - a.Lng)

	sinDLat := math.Sin(dLat / 2)
	sinDLng := math.Sin(dLng / 2)

	h := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLng*sinDLng
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
	return earthRadiusKM * c
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// Plan is the result of computing a delivery: how long it will take and
// whether the message is destined to be lost along the way.
type Plan struct {
	DistanceKM  float64
	SpeedKMH    float64
	Duration    time.Duration
	WillBeLost  bool
	DepartsAt   time.Time
	ArrivesAt   time.Time
}

// Compute builds a delivery Plan for a message sent from `from` to `to`
// using a carrier that travels at speedKMH, with lossProbability chance
// (0.0-1.0) of the message never arriving.
//
// A minimum duration floor is applied so very short distances (e.g. same
// building) still carry a few seconds of "flight" — part of the product's
// core value is the anticipation, not just physics.
func Compute(from, to Coordinates, speedKMH, lossProbability float64, now time.Time, rng *rand.Rand) Plan {
	dist := HaversineKM(from, to)

	hours := dist / speedKMH
	duration := time.Duration(hours * float64(time.Hour))

	const minDuration = 5 * time.Second
	if duration < minDuration {
		duration = minDuration
	}

	lost := rng.Float64() < lossProbability

	return Plan{
		DistanceKM: dist,
		SpeedKMH:   speedKMH,
		Duration:   duration,
		WillBeLost: lost,
		DepartsAt:  now,
		ArrivesAt:  now.Add(duration),
	}
}

// ApplySpeedup shortens the remaining time to arrival by pct percent
// (0-100), used when a user watches a rewarded ad. It never moves
// ArrivesAt earlier than "now".
func ApplySpeedup(plan Plan, pct float64, now time.Time) Plan {
	remaining := plan.ArrivesAt.Sub(now)
	if remaining <= 0 {
		plan.ArrivesAt = now
		return plan
	}
	cut := time.Duration(float64(remaining) * (pct / 100))
	plan.ArrivesAt = plan.ArrivesAt.Add(-cut)
	if plan.ArrivesAt.Before(now) {
		plan.ArrivesAt = now
	}
	return plan
}
