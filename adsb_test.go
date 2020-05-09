// go test -v github.com/skypies/adsb
package adsb

import (
	"testing"
)

func TestAdsbMsgGroundSpeed(t *testing.T) {
	m := Msg{}

	if m.HasGroundSpeed() {
		t.Errorf("ADS-B message should have no ground speed")
	}

	m.SetHasGroundSpeed()

	if !m.HasGroundSpeed() {
		t.Errorf("ADS-B message should have ground speed set")
	}
}

func TestAdsbTrack(t *testing.T) {
	m := Msg{}

	if m.HasTrack() {
		t.Errorf("ADS-B message should have no track")
	}

	m.SetHasTrack()

	if !m.HasTrack() {
		t.Errorf("ADS-B message should have track set")
	}
}

func TestAdsbPosition(t *testing.T) {
	m := Msg{}

	if m.HasPosition() {
		t.Errorf("ADS-B message should have no position")
	}

	m.SetHasPosition()

	if !m.HasPosition() {
		t.Errorf("ADS-B message should have position set")
	}
}

func TestAdsbAltitude(t *testing.T) {
	m := Msg{}

	if m.HasAltitude() {
		t.Errorf("ADS-B message should have no altitude")
	}

	m.SetHasAltitude()

	if !m.HasAltitude() {
		t.Errorf("ADS-B message should have altitude set")
	}
}

func TestAdsbCallsign(t *testing.T) {
	m := Msg{}

	if m.HasCallsign() {
		t.Errorf("ADS-B message should have no callsign")
	}

	m.SetHasCallsign()

	if !m.HasCallsign() {
		t.Errorf("ADS-B message should have callsign set")
	}
}

func TestAdsbVerticalRate(t *testing.T) {
	m := Msg{}

	if m.HasVerticalRate() {
		t.Errorf("ADS-B message should have no vertical rate")
	}

	m.SetHasVerticalRate()

	if !m.HasVerticalRate() {
		t.Errorf("ADS-B message should have vertical rate")
	}
}

func TestAdsbSquawk(t *testing.T) {
	m := Msg{}

	if m.HasSquawk() {
		t.Errorf("ADS-B message should have no squawk")
	}

	m.SetHasSquawk()

	if !m.HasSquawk() {
		t.Errorf("ADS-B message should have squawk set")
	}
}
