package delivery

import (
	"math/rand"
	"testing"
	"time"
)

func TestHaversineKM_SamePoint(t *testing.T) {
	a := Coordinates{Lat: -6.2, Lng: 106.8}
	d := HaversineKM(a, a)
	if d != 0 {
		t.Fatalf("expected 0 distance for same point, got %f", d)
	}
}

func TestHaversineKM_KnownDistance(t *testing.T) {
	// Jakarta to Surabaya, roughly 660-700km great circle.
	jakarta := Coordinates{Lat: -6.2088, Lng: 106.8456}
	surabaya := Coordinates{Lat: -7.2575, Lng: 112.7521}

	d := HaversineKM(jakarta, surabaya)
	if d < 600 || d > 700 {
		t.Fatalf("expected ~600-700km, got %f", d)
	}
}

func TestCompute_DurationScalesWithDistance(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	now := time.Now()

	near := Coordinates{Lat: -6.2, Lng: 106.8}
	far := Coordinates{Lat: 40.7, Lng: -74.0} // NYC

	nearPlan := Compute(near, Coordinates{Lat: -6.21, Lng: 106.81}, 177, 0, now, rng)
	farPlan := Compute(near, far, 177, 0, now, rng)

	if farPlan.Duration <= nearPlan.Duration {
		t.Fatalf("expected far plan to take longer: near=%v far=%v", nearPlan.Duration, farPlan.Duration)
	}
}

func TestCompute_MinimumDurationFloor(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	now := time.Now()
	same := Coordinates{Lat: -6.2, Lng: 106.8}

	plan := Compute(same, same, 177, 0, now, rng)
	if plan.Duration < 5*time.Second {
		t.Fatalf("expected minimum 5s floor, got %v", plan.Duration)
	}
}

func TestCompute_LossProbabilityZero(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()
	a := Coordinates{Lat: -6.2, Lng: 106.8}
	b := Coordinates{Lat: -7.2, Lng: 112.7}

	for i := 0; i < 100; i++ {
		plan := Compute(a, b, 177, 0, now, rng)
		if plan.WillBeLost {
			t.Fatalf("expected no loss with probability 0")
		}
	}
}

func TestCompute_LossProbabilityOne(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()
	a := Coordinates{Lat: -6.2, Lng: 106.8}
	b := Coordinates{Lat: -7.2, Lng: 112.7}

	plan := Compute(a, b, 177, 1.0, now, rng)
	if !plan.WillBeLost {
		t.Fatalf("expected loss with probability 1.0")
	}
}

func TestApplySpeedup_ReducesRemainingTime(t *testing.T) {
	now := time.Now()
	plan := Plan{
		DepartsAt: now,
		ArrivesAt: now.Add(10 * time.Minute),
	}

	updated := ApplySpeedup(plan, 50, now)
	remaining := updated.ArrivesAt.Sub(now)

	if remaining > 5*time.Minute+time.Second || remaining < 5*time.Minute-time.Second {
		t.Fatalf("expected ~5min remaining after 50%% speedup, got %v", remaining)
	}
}

func TestApplySpeedup_NeverGoesBeforeNow(t *testing.T) {
	now := time.Now()
	plan := Plan{
		DepartsAt: now.Add(-time.Minute),
		ArrivesAt: now.Add(-time.Second), // already arrived
	}

	updated := ApplySpeedup(plan, 50, now)
	if updated.ArrivesAt.Before(now) {
		t.Fatalf("ArrivesAt should not be before now")
	}
}
