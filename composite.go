package adsb

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
)

// CompositeMsg has the same data fields as a Msg, but contains data from multiple Msgs. This is
// because ADS-B messages are normally emitted with only some fields filled; to get altitude, speed
// location *and* callsign, you need to combine ~3 messages. We use a distinct type for these, to be
// unambiguous about where the data came from.
type CompositeMsg struct {
	Msg // Embedded stuct
	// Real UTC timefields ??
	ReceiverName  string // Some identifier for the ADS-B receiver that generated this data
}

// Need to differentiate from 'real' ADSB messages, and synthetic MLAT messages
func (cm CompositeMsg)DataSystem() string {
	switch cm.Type {
	case "MLAT": return "MLAT"
	case "MSG": return "ADSB"
	default: return cm.Type
	}
}

func (cm CompositeMsg)String() string {
	pos := fmt.Sprintf(" (%.7f,%.7f)", cm.Position.Lat, cm.Position.Long)
	return fmt.Sprintf("%s%d+ : %s[%7.7s] %5df, %3dk, %5df/m, %3ddeg, %s @ %s (%s) %s",
		cm.Type, cm.SubType, cm.Icao24, cm.Callsign,
		cm.Altitude, cm.GroundSpeed, cm.VerticalRate, cm.Track, pos, cm.GeneratedTimestampUTC,
		cm.ReceiverName, cm.DataSystem())
}

type CompositeMsgPtrByTimeAsc []*CompositeMsg
func (a CompositeMsgPtrByTimeAsc) Len() int          { return len(a) }
func (a CompositeMsgPtrByTimeAsc) Swap(i,j int)      { a[i],a[j] = a[j],a[i] }
func (a CompositeMsgPtrByTimeAsc) Less(i,j int) bool {
	return a[j].GeneratedTimestampUTC.After(a[i].GeneratedTimestampUTC)
}

func Base64EncodeMessages(msgs []*CompositeMsg) (string, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(msgs); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
	}
}

func Base64DecodeMessages(str string) ([]*CompositeMsg, error) {
	if data,err := base64.StdEncoding.DecodeString(str); err != nil {
		return nil,err
	} else {
		msgs := []*CompositeMsg{}
		buf := bytes.NewBuffer(data)
		err := gob.NewDecoder(buf).Decode(&msgs)
		return msgs, err
	}
}
