package app

import (
	"testing"
	"time"
)

func TestKeyTracker_DoubleG(t *testing.T) {
	kt := &KeyTracker{}

	// First 'g' should not trigger
	if kt.IsDoubleG('g') {
		t.Error("first g should not trigger double-g")
	}

	// Second 'g' within 300ms should trigger
	if !kt.IsDoubleG('g') {
		t.Error("second g should trigger double-g")
	}

	// After triggering, state should be reset
	if kt.IsDoubleG('g') {
		t.Error("third g should not trigger (state was reset)")
	}
}

func TestKeyTracker_NonGKeyResets(t *testing.T) {
	kt := &KeyTracker{}

	kt.IsDoubleG('g')
	kt.IsDoubleG('j') // different key resets

	if kt.IsDoubleG('g') {
		t.Error("g after non-g key should not trigger double-g")
	}
}

func TestKeyTracker_ExpiredTimeout(t *testing.T) {
	kt := &KeyTracker{}

	kt.IsDoubleG('g')
	// Simulate expiration
	kt.lastKeyTime = time.Now().Add(-500 * time.Millisecond)

	if kt.IsDoubleG('g') {
		t.Error("g after timeout should not trigger double-g")
	}
}
