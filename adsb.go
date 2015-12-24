package adsb

import (
	"fmt"
	"time"

	"github.com/skypies/geo"
)

type IcaoId string

// http://woodair.net/SBS/Article/Barebones42_Socket_Data.htm
// https://github.com/MalcolmRobb/dump1090/blob/master/mode_s.c#L834
//
// ** NOTE ** : we're not actually populating all of this yet
//
type Msg struct {
	// This set of data is basically the SBS1 format, not the ADS-B format.
	
	Type string // = 0 // type	 (MSG, STA, ID, AIR, SEL or CLK). We ignore all but MSG.
	SubType int64 // = 1 // Type	 MSG sub types 1 to 8. Not used by other message types.
	// Session = 2 // ID	 Database Session record number
	// AircraftID = 3 //	 Database Aircraft record number
	Icao24 IcaoId //  = 4 //	 Aircraft Mode S hexadecimal code
	GeneratedTimestampUTC time.Time
	LoggedTimestampUTC    time.Time
	//DateGen = 6 // message generated	  As it says
	//TimeGen = 7 // message generated	  As it says
	//DateLog = 8 // message logged	  As it says
	//TimeLog = 9 // message logged	  As it says
	Callsign string // = 10 //	 An eight digit flight ID - can be flight number or registration
	Altitude int64 // = 11 //	 Mode C altitude. Height relative to 1013.2mb (Flight Level).
	GroundSpeed int64 // = 12 //	 Speed over ground (not indicated airspeed)
	Track int64 // = 13 //	 Track of aircraft (not heading). Derived from the velocity E/W, N/S

	Position geo.Latlong
	//Latitude float64 // = 14 //	 North and East positive. South and West negative.
	//Longitude float64 // = 15 //	 North and East positive. South and West negative.

	VerticalRate int64 // = 16 //	 64ft resolution
	Squawk string // = 17 //	 Assigned Mode A squawk code.
	AlertSquawkChange bool // = 18 // (Squawk change)	 Flag to indicate squawk has changed.
	Emergency bool // = 19 //	 Flag to indicate emergency code has been set
	SPI bool // = 20 // (Ident)	 Flag to indicate transponder Ident has been activated.
	IsOnGround bool // = 21 //	 Flag to indicate ground squat switch is active

	// Flags filled (and only valid) during parsing, interrogated by methods below
	hasPosition bool
}

func (m Msg)HasPosition() bool { return m.hasPosition }

func (m Msg)String() string {
	s := fmt.Sprintf("%s%d : %s", m.Type, m.SubType, m.Icao24)
	if m.HasPosition() {
		s += fmt.Sprintf(" %s", m.Position)
	}
	return s
}

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


// CompositeMsg has the same data fields as a Msg, but contains data from multiple Msgs. This is
// because ADS-B messages are normally emitted with only some fields filled; to get altitude, speed
// location *and* callsign, you need to combine ~3 messages. We use a distinct type for these, to be
// unambiguous about where the data came from.
type CompositeMsg struct {
	Msg // Embedded stuct
	// Real UTC timefields ??
	ReceiverName  string // Some identifier for the ADS-B receiver that generated this data
}

func (cm CompositeMsg)String() string {
	pos := fmt.Sprintf(" (%.7f,%.7f)", cm.Position.Lat, cm.Position.Long)
	return fmt.Sprintf("%s%d+ : %s[%7.7s] %5df, %3dk, %5df/m, %3ddeg, %s @ %s (%s)",
		cm.Type, cm.SubType, cm.Icao24, cm.Callsign,
		cm.Altitude, cm.GroundSpeed, cm.VerticalRate, cm.Track, pos, cm.GeneratedTimestampUTC,
		cm.ReceiverName)
}

type CompositeMsgPtrByTimeAsc []*CompositeMsg
func (a CompositeMsgPtrByTimeAsc) Len() int          { return len(a) }
func (a CompositeMsgPtrByTimeAsc) Swap(i,j int)      { a[i],a[j] = a[j],a[i] }
func (a CompositeMsgPtrByTimeAsc) Less(i,j int) bool {
	return a[j].GeneratedTimestampUTC.After(a[i].GeneratedTimestampUTC)
}
