// TrackBuffer accumulates ADSB messages, grouped by aircraft, and flushes out
// bundles of them.
package trackbuffer

import (
	"sort"
	"time"
	"github.com/skypies/adsb"
)

// A slice of ADSB messages that share the same IcaoId
type Track struct {
	Messages  []*adsb.CompositeMsg
}

type TrackBuffer struct {
	MaxAge      time.Duration // Flush any track with data older than this
	Tracks      map[adsb.IcaoId]*Track
}

func NewTrackBuffer() *TrackBuffer {
	tb := TrackBuffer{
		MaxAge: time.Second*30,
		Tracks: make(map[adsb.IcaoId]*Track),
	}
	return &tb
}

func (t *Track)Age() time.Duration {
	if len(t.Messages)==0 { return time.Duration(time.Hour * 24) }
	return time.Since(t.Messages[0].GeneratedTimestampUTC)
}

func (tb *TrackBuffer)AddTrack(icao adsb.IcaoId) {
	track := Track{
		Messages: []*adsb.CompositeMsg{},
	}
	tb.Tracks[icao] = &track
}

func (tb *TrackBuffer)RemoveTracks(icaos []adsb.IcaoId) []*Track{
	removed := []*Track{}
	for _,icao := range icaos {
		removed = append(removed, tb.Tracks[icao])
		delete(tb.Tracks, icao)
	}
	return removed
}

func (tb *TrackBuffer)AddMessage(m *adsb.CompositeMsg) {
	if _,exists := tb.Tracks[m.Icao24]; exists == false {
		tb.AddTrack(m.Icao24)
	}
	track := tb.Tracks[m.Icao24]
	track.Messages = append(track.Messages, m)
}

func (tb *TrackBuffer)Flush(flushChan chan<- []*adsb.CompositeMsg) {
	toRemove := []adsb.IcaoId{}
	
	for id,_ := range tb.Tracks {
		if tb.Tracks[id].Age() > tb.MaxAge {
			toRemove = append(toRemove, id)
		}
	}

	for _,t := range tb.RemoveTracks(toRemove) {
		sort.Sort(adsb.CompositeMsgPtrByTimeAsc(t.Messages))
		flushChan <- t.Messages
	}
}
