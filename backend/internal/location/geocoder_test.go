package location

import "testing"

func TestSameCity(t *testing.T) {
	if !SameCity(" Surabaya ", "surabaya") {
		t.Fatal("expected normalized city names to match")
	}
	if SameCity("Malang", "Surabaya") {
		t.Fatal("expected different cities not to match")
	}
	if SameCity("", "Surabaya") {
		t.Fatal("empty city must not match")
	}
}
