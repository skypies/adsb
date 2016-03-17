/* Package msgbuffer is a temporary buffer of ADS-B messages.

It uses a strict whitelist approach, so that it can immediately
discard irrelevant messages.

It caches useful data from previous messages, and generates
'composite' output messages with full data (i.e. every output packet
gets a callsign).

When a maximum age limit is reached, the slice of accumulated messages
are sent down a channel.

It contains enough memory housekeeping to be used indefinitely.

Sample usage:

    mb := msgbuffer.NewMsgBuffer()
    mb.MaxMessageAge      = time.Second * 0  // How long to wait before flushing; 0==no wait
    mb.MinPublishInterval = time.Second * 0  // How long must wait between flushes; 0==no wait
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
	MinPublishInterval time.Duration  // Regardless of all else, don't publish faster than this
	MaxMessageAge      time.Duration  // If we've held a message for more than this, flush the buffer
	MaxQuietTime       time.Duration  // If a sender sends no messages for this long, remove it

	Senders            map[adsb.IcaoId]*ADSBSender // Alive things we're currently getting data from
	Messages        []*adsb.CompositeMsg           // The actual buffer of messages

	FlushChannel       chan<- []*adsb.CompositeMsg
	lastFlush          time.Time
	lastAgeOut         time.Time
}

func (mb MsgBuffer)String() string {
	s := fmt.Sprintf("--{ MsgBuffer (maxage=%s, maxwait=%s, minpub=%s) }--\n",
		mb.MaxMessageAge, mb.MaxQuietTime, mb.MinPublishInterval)
	for k,sender := range mb.Senders { s += fmt.Sprintf(" - %s %s\n", k, sender) }
	for i,m := range mb.Messages { s += fmt.Sprintf("[%02d] %s\n", i, m) }
	return s
}

// }}}

// {{{ NewMsgBuffer

func NewMsgBuffer() *MsgBuffer {
	return &MsgBuffer{
		MinPublishInterval:  time.Second * 5,
		MaxMessageAge:       time.Second * 30,
		MaxQuietTime:        time.Second * 360,
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

	// If the message had any of the optional fields, cache the value for later
	if m.HasCallsign()      {
		if len(m.Callsign) > 0 {
			s.LastCallsign = m.Callsign
		} else {
			s.LastCallsign = "_._._._." // Our nil value :/
		}
	}
	if m.HasSquawk()        { s.LastSquawk        = m.Squawk }
	if m.HasGroundSpeed()   { s.LastGroundSpeed   = m.GroundSpeed }
	if m.HasTrack()         { s.LastTrack         = m.Track }
	if m.HasVerticalRate()  { s.LastVerticalSpeed = m.VerticalRate }
	
	if m.Type == "MSG_foooo" {
		if m.SubType == 1 {
			// TODO: move this to m.hasCallsign()
			// MSG,1 - the callsign/ident subtype - is sometimes blank. But we
			// don't really want to confuse flights that have a purposefully
			// blank callsign with those for yet we've yet to receive a MSG,1.
			// So we use a magic string instead.
			if m.Callsign == ""     { s.LastCallsign      = "_._._._." }
			if m.Callsign != ""     { s.LastCallsign      = m.Callsign }
		
		} else if m.SubType == 2 {
			if m.HasGroundSpeed()   { s.LastGroundSpeed   = m.GroundSpeed }
			if m.HasTrack()         { s.LastTrack         = m.Track }

		} else if m.SubType == 4 {
			if m.HasGroundSpeed()   { s.LastGroundSpeed   = m.GroundSpeed }
			if m.HasVerticalRate()  { s.LastVerticalSpeed = m.VerticalRate }
			if m.HasTrack()         { s.LastTrack         = m.Track }

		} else if m.SubType == 6 {
			if m.Squawk != ""       { s.LastSquawk        = m.Squawk }
		}
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

	//if s.LastGroundSpeed == 0 || s.LastTrack == 0 || s.LastCallsign == "" { return nil }

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
		if time.Since(mb.Senders[id].LastSeen) >= mb.MaxQuietTime {
			delete(mb.Senders, id)
			removed++
		}
	}

	return
}

// }}}
// {{{ MsgBuffer.flush

func (mb *MsgBuffer)flush() {
	if mb.FlushChannel != nil {
		mb.FlushChannel <- mb.Messages
	}

	// Reset the accumulator
	mb.Messages = []*adsb.CompositeMsg{}
	mb.lastFlush = time.Now()
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

	// We use the timestamp in the message to decide when to flush,
	// rather than the time at which we received the message; this is to
	// deliver a better end-to-end QoS for message delivery.

	// But stale messages can arrive, with timestamps from the past;
	// they would always trigger a flush, and flushing every message
	// slows things down (so we never ever catch up again :().
	// So we also enforce a minimum interval between flushes.
	if len(mb.Messages) > 0 {
		t := mb.Messages[0].GeneratedTimestampUTC
		if time.Since(t) >= mb.MaxMessageAge && time.Since(mb.lastFlush) >= mb.MinPublishInterval {
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
