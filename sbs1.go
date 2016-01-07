package adsb

import (
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"github.com/skypies/geo"
	//"github.com/skypies/date"
)
// http://woodair.net/SBS/Article/Barebones42_Socket_Data.htm
const (
	SBS1Message = 0 // type	 (MSG, STA, ID, AIR, SEL or CLK)
	SBS1Transmission = 1 // Type	 MSG sub types 1 to 8. Not used by other message types.
	SBS1Session = 2 // ID	 Database Session record number
	SBS1AircraftID = 3 //	 Database Aircraft record number
	SBS1Icao24 = 4 //	 Aircraft Mode S hexadecimal code
	SBS1FlightID = 5 //	 Database Flight record number
	SBS1DateGen = 6 // message generated	  As it says
	SBS1TimeGen = 7 // message generated	  As it says
	SBS1DateLog = 8 // message logged	  As it says
	SBS1TimeLog = 9 // message logged	  As it says
	SBS1Callsign = 10 //	 An eight digit flight ID - can be flight number or registration (or even nothing).
	SBS1Altitude= 11 //	 Mode C altitude. Height relative to 1013.2mb (Flight Level). Not height AMSL..
	SBS1GroundSpeed = 12 //	 Speed over ground (not indicated airspeed)
	SBS1Track = 13 //	 Track of aircraft (not heading). Derived from the velocity E/W and velocity N/S
	SBS1Latitude = 14 //	 North and East positive. South and West negative.
	SBS1Longitude = 15 //	 North and East positive. South and West negative.
	SBS1VerticalRate = 16 //	 64ft resolution
	SBS1Squawk = 17 //	 Assigned Mode A squawk code.
	SBS1AlertSquawkChange = 18 // (Squawk change)	 Flag to indicate squawk has changed.
	SBS1Emergency = 19 //	 Flag to indicate emergency code has been set
	SBS1SPI = 20 // (Ident)	 Flag to indicate transponder Ident has been activated.
	SBS1IsOnGround = 21 //	 Flag to indicate ground squat switch is active
)

// Hack global. Maybe should have a parser struct. Default is for backwards compatibility.
var TimeLocation = "America/Los_Angeles"  

func toTime(d,t string) (time.Time, error) {
	format := "2006/01/02 15:04:05.999999999"
	value := d+" "+t
	if loc, err := time.LoadLocation(TimeLocation); err != nil {
		panic(err)
	} else if t, err := time.ParseInLocation(format, value, loc); err != nil {
		return time.Now(), err
	} else {
		return t.UTC(), nil
	}
}

func (m *Msg)FromSBS1(s string) error {
	ioReader := strings.NewReader(s)
	csvReader := csv.NewReader(ioReader)
	if r,err := csvReader.Read(); err != nil {
		return err
	} else {
		// Dear god. If any block of code ever needed try/catch ...
		m.Type = r[SBS1Message]
		if i,err := strconv.ParseInt(r[SBS1Transmission], 10, 64); err != nil {
			return err
		} else {
			m.SubType = i
		}
		m.Icao24 = IcaoId(r[SBS1Icao24])
		
		if t,err := toTime(r[SBS1DateGen], r[SBS1TimeGen]); err != nil {
			return err
		} else {			
			m.GeneratedTimestampUTC = t
		}
		if t,err := toTime(r[SBS1DateLog], r[SBS1TimeLog]); err != nil {
			return err
		} else {
			m.LoggedTimestampUTC = t
		}

		m.Callsign = strings.TrimSpace(r[SBS1Callsign])
		m.Squawk = strings.TrimSpace(r[SBS1Squawk])

		if (r[SBS1Altitude] != "") {
			if i,err := strconv.ParseInt(r[SBS1Altitude], 10, 64); err != nil {
				return err
			} else {
				m.Altitude = i
			}
		}
		if (r[SBS1GroundSpeed] != "") {
			if i,err := strconv.ParseInt(r[SBS1GroundSpeed], 10, 64); err != nil {
				return err
			} else {
				m.GroundSpeed = i
			}
		}
		if (r[SBS1VerticalRate] != "") {
			if i,err := strconv.ParseInt(r[SBS1VerticalRate], 10, 64); err != nil {
				return err
			} else {
				m.VerticalRate = i
			}
		}
		if (r[SBS1Track] != "") {
			if i,err := strconv.ParseInt(r[SBS1Track], 10, 64); err != nil {
				return err
			} else {
				m.Track = i
			}
		}
		
		// Shoud prob decide this based on message type.
		if (r[SBS1Latitude] != "" && r[SBS1Longitude] != "") {
			if lat,err := strconv.ParseFloat(r[SBS1Latitude], 64); err != nil {
				return err
			} else if long,err := strconv.ParseFloat(r[SBS1Longitude], 64); err != nil {
				return err
			} else {
				m.Position = geo.Latlong{lat, long}
				m.hasPosition = true
			}
		}
	}
	return nil
}
