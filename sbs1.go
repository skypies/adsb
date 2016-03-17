package adsb

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/skypies/geo"
	//"github.com/skypies/util/date"
)

// http://woodair.net/SBS/Article/Barebones42_Socket_Data.htm
// https://github.com/mutability/mlat-client/blob/master/mlat/client/output.py#L196
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

	// The extra fields for ext_basestation (used for MLAT output)
	// https://github.com/mutability/mlat-client/blob/master/mlat/client/output.py#L264
	ExtSBSNumStations = 22
	// 23 left blank for now
	ExtSBSErrorEstimate = 24
)

// Hack global. Maybe should have a parser struct.
var TimeLocation = "UTC" // "America/Los_Angeles"  

func toTimeUTC(d,t string) (time.Time, error) {
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

		// ext_basestation format has 25 fields ...
		if len(r) != 22 && len(r) != 25 {
			return fmt.Errorf("Message was corrupt; has %d fields", len(r))
		}

		// Dear god. If any block of code ever needed try/catch ...
		m.Type = r[SBS1Message]
		if i,err := strconv.ParseInt(r[SBS1Transmission], 10, 64); err != nil {
			return err
		} else {
			m.SubType = i
		}
		m.Icao24 = IcaoId(r[SBS1Icao24])
		
		if t,err := toTimeUTC(r[SBS1DateGen], r[SBS1TimeGen]); err != nil {
			return err
		} else {			
			m.GeneratedTimestampUTC = t
		}
		if t,err := toTimeUTC(r[SBS1DateLog], r[SBS1TimeLog]); err != nil {
			return err
		} else {
			m.LoggedTimestampUTC = t
		}

		if len(r[SBS1Callsign]) > 0 {
			m.hasCallsign = true
			m.Callsign = strings.TrimSpace(r[SBS1Callsign]) // This may truncate to the empty string.
		}
		if len(r[SBS1Squawk]) > 0 {
			m.hasSquawk = true
			m.Squawk = strings.TrimSpace(r[SBS1Squawk])
		}
		
		if (r[SBS1Altitude] != "") {
			if i,err := strconv.ParseInt(r[SBS1Altitude], 10, 64); err != nil {
				return err
			} else {
				m.Altitude = i
				m.hasAltitude = true
			}
		}
		if (r[SBS1GroundSpeed] != "") {
			if i,err := strconv.ParseInt(r[SBS1GroundSpeed], 10, 64); err != nil {
				return err
			} else {
				m.GroundSpeed = i
				m.hasGroundSpeed = true
			}
		}
		if (r[SBS1Track] != "") {
			if i,err := strconv.ParseInt(r[SBS1Track], 10, 64); err != nil {
				return err
			} else {
				m.Track = i
				m.hasTrack = true
			}
		}
		if (r[SBS1VerticalRate] != "") {
			if i,err := strconv.ParseInt(r[SBS1VerticalRate], 10, 64); err != nil {
				return err
			} else {
				m.VerticalRate = i
				m.hasVerticalRate = true
			}
		}
		
		// Shoud prob decide this based on message type.
		if (r[SBS1Latitude] != "" && r[SBS1Longitude] != "") {
			if lat,err := strconv.ParseFloat(r[SBS1Latitude], 64); err != nil {
				return err
			} else if long,err := strconv.ParseFloat(r[SBS1Longitude], 64); err != nil {
				return err
			} else {//if lat!=0.0 && long>0.0 { // Some dodgy data outputs nil locations as "0.00,0.00"
				m.Position = geo.Latlong{lat, long}
				m.hasPosition = true
			}
		}

		// Extended basestation format ?
		if len(r) == 25 {
			if (r[ExtSBSNumStations] != "") {
				if i,err := strconv.ParseInt(r[ExtSBSNumStations], 10, 64); err != nil {
					m.NumStations = i
				}
			}
			// Not sure what this data type is, omit for now
			//if (r[ExtSBSErrorEstimate] != "") {
			//	if i,err := strconv.ParseInt(r[ExtSBSErrorEstimate], 10, 64); err != nil {
			//		m.ErrorEstimate = i
			//	}
			//}
		}
	}
	return nil
}

func (m *Msg)ToSBS1() string {
	r := make([]string, 22)

	r[SBS1Message]      = m.Type
	r[SBS1Transmission] = fmt.Sprintf("%d", m.SubType)
	//r[SBS1Session]      = "" // = 2 // ID	 Database Session record number
	//r[SBS1AircraftID] = "" // = 3 //	 Database Aircraft record number
	r[SBS1Icao24]       = string(m.Icao24)
	//r[SBS1FlightID] = "" // = 5 //	 Database Flight record number
	r[SBS1DateGen]      = m.GeneratedTimestampUTC.Format("2006/01/02")
	r[SBS1TimeGen]      = m.GeneratedTimestampUTC.Format("15:04:05.999999999")
	r[SBS1DateLog]      = m.LoggedTimestampUTC.Format("2006/01/02")
	r[SBS1TimeLog]      = m.LoggedTimestampUTC.Format("15:04:05.999999999")
	r[SBS1Callsign]     = m.Callsign
	r[SBS1Altitude]     = fmt.Sprintf("%d", m.Altitude)
	r[SBS1GroundSpeed]  = fmt.Sprintf("%d", m.GroundSpeed)
	r[SBS1Track]        = fmt.Sprintf("%d", m.Track)

	if m.HasPosition() {
		r[SBS1Latitude]     = fmt.Sprintf("%.5f", m.Position.Lat)  // Too much precision ?
		r[SBS1Longitude]    = fmt.Sprintf("%.5f", m.Position.Long)
	}
	r[SBS1VerticalRate] = fmt.Sprintf("%d", m.VerticalRate)
	r[SBS1Squawk]       = m.Squawk
	//r[SBS1AlertSquawkChange] = "" // = 18 // (Squawk change)	 Flag to indicate squawk has changed.
	//r[SBS1Emergency] = "" // = 19 //	 Flag to indicate emergency code has been set
	//r[SBS1SPI] = "" // = 20 // (Ident)	 Flag to indicate transponder Ident has been activated.
	//r[SBS1IsOnGround] = "" // = 21 //	 Flag to indicate ground squat switch is active

	// May need to do something here ...
	if m.IsMLAT() {}
	
	return strings.Join(r, ",")
}
