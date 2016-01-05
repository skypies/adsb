/* Package msgbuffer is a temporary buffer of ADS-B messages.

It uses a strict whitelist approach, so that it can immediately
discard irrelevant messages.

It caches useful data from previous messages, and generates
'composite' output messages with full data (i.e. every output packet
gets a callsign).

When the messages are at risk of getting stale, it will invoke the
supplied callback to flush them.

It contains enough memory housekeeping to be used indefinitely.

Sample usage:

    mb := msgbuffer.NewMsgBuffer()
    mb.MaxMessageAgeSeconds = 0  // How long to wait before flushing; 0==no wait
    mb.FlushFunc = func(msgs []*adsb.CompositeMsg) {
      fmt.Printf("Just flushed %d messages\n", len(msgs))
    }
    
    myMessages := []adsb.Message{ ... }
    for _,msg := range myMessages {
      mb.Add(msg}
    }

*/
package msgbuffer

import(
	"fmt"
	"time"
	"github.com/skypies/adsb"
)

// {{{ ADSBSender{}

// ADSBSender stores some data that we use to flesh out partial messages
type ADSBSender struct {
	LastSeen          time.Time
	LastGroundSpeed   int64
	LastVerticalSpeed int64
	LastTrack         int64
	LastCallsign      string
	LastSquawk        string
}

func (s ADSBSender)String() string {
	return fmt.Sprintf("[%-7.7s],[%s] % 3dk, % 5df/m, % 3ddeg @ %s [%s]",
		s.LastCallsign, s.LastSquawk, s.LastGroundSpeed, s.LastVerticalSpeed, s.LastTrack, s.LastSeen,
		time.Since(s.LastSeen))
}

// }}}
// {{{ MsgBuffer{}

type MsgBuffer struct {
	FlushFunc func([]*adsb.CompositeMsg)

	MaxMessageAgeSeconds int64   // If we've held a message for more than this, flush the buffer
	MaxQuietTimeSeconds int64 // If an aircraft sends no messages for more than this, remove it
	Senders    map[adsb.IcaoId]*ADSBSender // Which senders we're currently getting data from

	Messages []*adsb.CompositeMsg  // The actual buffer

	lastAgeOut time.Time
}

func (mb MsgBuffer)String() string {
	s := fmt.Sprintf("--{ MsgBuffer (maxage=%d, maxwait=%d) }--\n",
		mb.MaxMessageAgeSeconds, mb.MaxQuietTimeSeconds)
	for k,sender := range mb.Senders { s += fmt.Sprintf(" - %s %s\n", k, sender) }
	for i,m := range mb.Messages { s += fmt.Sprintf("[%02d] %s\n", i, m) }
	return s
}

// }}}

// {{{ NewMsgBuffer

func NewMsgBuffer() *MsgBuffer {
	dumbFunc := func(msgs []*adsb.CompositeMsg) {
		fmt.Printf("MsgBuffer flushed %d messages (default FlushFunc)\n", len(msgs))
	}
	return &MsgBuffer{
		FlushFunc: dumbFunc,
		MaxMessageAgeSeconds: 30,
		MaxQuietTimeSeconds: 360,
		Senders: make(map[adsb.IcaoId]*ADSBSender),
	}
}

// }}}

// {{{ ADSBSender.updateFromMsg

// Some subtype packets have data we don't get in the bulk of position packets (those of subtype:3),
// so just cache their interesting data and inject it into next position packet.
// http://woodair.net/SBS/Article/Barebones42_Socket_Data.htm
func (s *ADSBSender)updateFromMsg(m *adsb.Msg) {
	s.LastSeen = time.Now().UTC()
	if m.SubType == 1 {
		// MSG,1 - the callsign/ident subtype - is sometimes blank. But we
		// don't really want to confuse flights that have a purposefully
		// blank callsign with those for yet we've yet to receive a MSG,1.
		// So we use a magic string instead.
		if m.Callsign == ""     { s.LastCallsign      = "_._._._." }
		if m.Callsign != ""     { s.LastCallsign      = m.Callsign }
		
	} else if m.SubType == 2 {
		if m.GroundSpeed != 0   { s.LastGroundSpeed   = m.GroundSpeed }
		if m.Track != 0         { s.LastTrack         = m.Track }

	} else if m.SubType == 4 {
		if m.GroundSpeed != 0   { s.LastGroundSpeed   = m.GroundSpeed }
		if m.VerticalRate != 0  { s.LastVerticalSpeed = m.VerticalRate }
		if m.Track != 0         { s.LastTrack         = m.Track }

	} else if m.SubType == 6 {
		if m.Squawk != ""       { s.LastSquawk        = m.Squawk }
  }
}

// }}}
// {{{ ADSBSender.maybeCreateComposite

// If this message has new position info, *and* we have good backfill, then craft a CompositeMsg.
// Note, we don't wait for squawk info.
func (s *ADSBSender)maybeCreateComposite(m *adsb.Msg) *adsb.CompositeMsg {
	if !m.HasPosition() {
		return nil
	}
	if s.LastGroundSpeed == 0 || s.LastVerticalSpeed == 0 || s.LastTrack == 0 || s.LastCallsign == "" {
		return nil
	}

	cm := adsb.CompositeMsg{Msg:*m}  // Clone the input into the embedded struct

	// Overwrite with cached info (from previous packets), if we don't have it in this packet
	if cm.GroundSpeed == 0  { cm.GroundSpeed  = s.LastGroundSpeed }
	if cm.VerticalRate == 0 { cm.VerticalRate = s.LastVerticalSpeed }
	if cm.Track == 0        { cm.Track        = s.LastTrack }
	if cm.Callsign == ""    { cm.Callsign     = s.LastCallsign }
	if cm.Squawk == ""      { cm.Squawk       = s.LastSquawk }
	
	return &cm
}

// }}}

// {{{ MsgBuffer.ageOutQuietSenders

func (mb *MsgBuffer)ageOutQuietSenders() (removed int64) {
	if time.Since(mb.lastAgeOut) < time.Second { return } // Only run once per second.
	mb.lastAgeOut = time.Now()

	for id,_ := range mb.Senders {
		if time.Since(mb.Senders[id].LastSeen) >= time.Second * time.Duration(mb.MaxQuietTimeSeconds) {
			delete(mb.Senders, id)
			removed++
		}
	}

	return
}

// }}}
// {{{ MsgBuffer.flush

func (mb *MsgBuffer)flush() {
	msgsToFlush := mb.Messages

	mb.Messages = []*adsb.CompositeMsg{}
	// Now, the only reference to the messages we're flushing is in msgsToFlush

	mb.FlushFunc(msgsToFlush) // Spawn goroutine here ? Or expect caller to handle all that ?
}

// }}}

// {{{ MsgBuffer.Add

// MaybeAdd looks at a new message, and updates the buffer as appropriate.
func (mb *MsgBuffer)Add(m *adsb.Msg) {

	mb.ageOutQuietSenders()
	
	if _,exists := mb.Senders[m.Icao24]; exists == false {
		// We've not seen this sender before. If we have position data,
		// start the whitelisting thing. We only Whitelist senders who
		// will eventually send useful info (e.g. position), so wait until
		// we see that.
		if m.HasPosition() {
			mb.Senders[m.Icao24] = &ADSBSender{LastSeen: time.Now().UTC()}
		}
	} else {
		mb.Senders[m.Icao24].updateFromMsg(m) // Pluck out anything interesting
		if composite := mb.Senders[m.Icao24].maybeCreateComposite(m); composite != nil {
			// We have a message to store !!
			mb.Messages = append(mb.Messages, composite)
		}
	}

	if len(mb.Messages) > 0 {
		t := mb.Messages[0].GeneratedTimestampUTC
		if time.Since(t) >= (time.Second * time.Duration(mb.MaxMessageAgeSeconds)) {
			mb.flush()
		}
	}
}

// }}}
// {{{ MsgBuffer.FinalFlush

func (mb *MsgBuffer)FinalFlush() {
	mb.flush()
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
