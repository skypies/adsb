package adsb

import (
	"fmt"
	"strings"
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
	
	Type string //`json:"-"` // = 0 // type	 (MSG, STA, ID, AIR, SEL or CLK). We ignore all but MSG.
	SubType int64 `json:"-"` // = 1 // Type	 MSG sub types 1 to 8. Not used by other message types.
	// Session = 2 // ID	 Database Session record number
	// AircraftID = 3 //	 Database Aircraft record number
	Icao24 IcaoId //  = 4 //	 Aircraft Mode S hexadecimal code

	////
	// NOTE - these are only going to be in UTC iff you've set adsb.TimeLocation to agree with
	// localtime on the machine running dump1090, which outputs non-timezoned 'local' time data.
	////
	GeneratedTimestampUTC time.Time
	LoggedTimestampUTC    time.Time `json:"-"`

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
	AlertSquawkChange bool `json:"-"`// = 18 // (Squawk change)	 Flag to indicate squawk has changed.
	Emergency bool `json:"-"`// = 19 //	 Flag to indicate emergency code has been set
	SPI bool `json:"-"`// = 20 // (Ident)	 Flag to indicate transponder Ident has been activated.
	IsOnGround bool `json:"-"`// = 21 //	 Flag to indicate ground squat switch is active

	// These fields are present for extended basestation format messages (i.e. MLAT)
	NumStations int64 `json:"-"`
	//ErrorEstimate int64 `json:"-"`  // Not sure if this is a float or an int, or what it means
	
	// Flags filled (and only valid) during initial SBS parsing, for fields not
	// always present
	hasAltitude     bool
	hasCallsign     bool
	hasSquawk       bool
	hasGroundSpeed  bool
	hasTrack        bool
	hasPosition     bool
	hasVerticalRate bool
}

func (m Msg)IsMLAT() bool { return m.Type == "MLAT" }

func (m Msg)IsMasked() bool { return strings.HasPrefix(string(m.Icao24), "~") }

//func (m Msg)HasAltitude()     bool { return m.hasAltitude }
func (m Msg)HasCallsign()     bool { return m.hasCallsign }
func (m Msg)HasSquawk()       bool { return m.hasSquawk }
func (m Msg)HasGroundSpeed()  bool { return m.hasGroundSpeed }
func (m Msg)HasTrack()        bool { return m.hasTrack }
func (m Msg)HasPosition()     bool { return m.hasPosition }
func (m Msg)HasVerticalRate() bool { return m.hasVerticalRate }

func (m Msg)String() string {
	s := fmt.Sprintf("%s%d : %s", m.Type, m.SubType, m.Icao24)
	if m.HasPosition() {
		s += fmt.Sprintf(" %s", m.Position)
	}
	return s
}
