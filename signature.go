package adsb

import (
	"github.com/skypies/geo"
)

// Signature is a subset of a composite ADSB message that can be considered
// to identify the content of the message; if two messages have equivalent
// Signatures, then we can consider them to be identical / duplicates.
type Signature struct {
	Pos geo.Latlong
	Icao24 IcaoId
}

func (m *CompositeMsg)GetSignature() Signature {
	return Signature{
		Pos: m.Position,
		Icao24: m.Icao24,
	}
}
