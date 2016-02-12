// go test -v github.com/skypies/adsb/msgbuffer
package msgbuffer

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/skypies/adsb"
)

func msgs(sbs string) (ret []adsb.Msg) {
	scanner := bufio.NewScanner(strings.NewReader(sbs))
	for scanner.Scan() {
		m := adsb.Msg{}
		text := scanner.Text()
		if err := m.FromSBS1(text); err != nil {
			panic(err)
		}
		m.GeneratedTimestampUTC = time.Now()  // Fake this out
		ret = append(ret, m)
	}
	return
}

var (
	maybeAddSBS = `MSG,7,1,1,A81BD0,1,2015/11/27,21:31:02.722,2015/11/27,21:31:02.721,,20150,,,,,,,,,,0
MSG,3,1,1,A81BD0,1,2015/11/27,21:31:03.354,2015/11/27,21:31:03.316,,20125,,,36.69804,-121.86007,,,,,,0
MSG,3,1,1,A81BD0,1,2015/11/27,21:31:03.704,2015/11/27,21:31:03.716,,20125,,,36.69830,-121.86017,,,,,,0
MSG,4,1,1,A81BD0,1,2015/11/27,21:31:04.704,2015/11/27,21:31:04.689,,,304,328,,,-1856,,,,,0
MSG,7,1,1,A81BD0,1,2015/11/27,21:31:04.753,2015/11/27,21:31:04.752,,20100,,,,,,,,,,0
MSG,1,1,1,A81BD0,1,2015/11/27,21:31:05.205,2015/11/27,21:31:05.153,VRD961  ,,,,,,,,,,,0
MSG,6,1,1,A81BD0,1,2015/11/27,21:31:05.255,2015/11/27,21:31:05.253,,,,,,,,1200,-1,0,0,0
MSG,3,1,1,A81BD0,1,2015/11/27,21:31:05.274,2015/11/27,21:31:05.276,,20075,,,36.70029,-121.86190,,,,,,0`

	unrelatedSBS = `MSG,7,1,1,ABEEF0,1,2015/11/27,21:31:04.753,2015/11/27,21:31:04.752,,20100,,,,,,,,,,0`
)

func TestAdd(t *testing.T) {
	m := msgs(maybeAddSBS)
	mb := NewMsgBuffer()

	mb.Add(&m[0])
	if len(mb.Senders) != 0 { t.Errorf("accepted boring packet as new sender") }
	
	mb.Add(&m[1])
	if len(mb.Senders) != 1 { t.Errorf("did not add pos packet as new sender") }
	if len(mb.Messages) != 0 { t.Errorf("added pos packet as output") }

	// Cache a pointer to the sender record, so we can interrogate it as things unfold
	var id adsb.IcaoId
	for k,_ := range mb.Senders { id = k }
	
	mb.Add(&m[2])
	if len(mb.Messages) != 0 { t.Errorf("added pos packet (type 3) as output") }
	
	
	mb.Add(&m[3])
	if len(mb.Messages) != 0 { t.Errorf("added speed packet (type 4) as output") }
	if mb.Senders[id].LastGroundSpeed == 0 { t.Errorf("no sender update from speed packet") }

	mb.Add(&m[4])
	if len(mb.Messages) != 0 { t.Errorf("added altitude packet (type 7) as output") }
	// We don't bother to update sender info for altitude as every position packet comes with altitude data

	mb.Add(&m[5])
	if len(mb.Messages) != 0 { t.Errorf("added callsign packet (type 4) as output") }
	if mb.Senders[id].LastCallsign == "" { t.Errorf("no sender update from callsign packet") }

	mb.Add(&m[6])
	if len(mb.Messages) != 0 { t.Errorf("added squawk packet (type 6) as output") }
	if mb.Senders[id].LastSquawk == "" { t.Errorf("no sender update from squawk packet") }

	// Now we have a callsign, pos packets should get emitted into buffer

	mb.Add(&m[7])
	if len(mb.Messages) != 1 { t.Errorf("post-callsign pos packet not emitted") }
	
	// fmt.Printf("%s", mb)
}

func TestAgeOutQuietSenders(t *testing.T) {
	mb := NewMsgBuffer()
	messages := msgs(maybeAddSBS)
	for _,msg := range messages {
		mb.Add(&msg)
	}

	unrelatedMsg := adsb.Msg{}
	if err := unrelatedMsg.FromSBS1(unrelatedSBS); err != nil { panic(err) }
	
	// Pluck out the (only) sender ID, reset its clock into the past
	var id adsb.IcaoId
	for k,_ := range mb.Senders { id = k }

	// Rig time - just before the age out window
	offset := -1 * mb.MaxQuietTime - 5
	mb.Senders[id].LastSeen = mb.Senders[id].LastSeen.Add(offset)
	mb.Add(&unrelatedMsg) // Send a message, to trigger ageout
	if len(mb.Senders) != 1 { t.Errorf("aged out too soon ?") }

	// Rig time - just after the age out window. And reset the sweep time.
	mb.Senders[id].LastSeen = mb.Senders[id].LastSeen.Add(time.Duration(-10) * time.Second)
	mb.lastAgeOut = mb.lastAgeOut.Add(time.Second * time.Duration(-5))
	mb.Add(&unrelatedMsg) // Send a message, to trigger ageout
	if len(mb.Senders) != 0 { t.Errorf("aged out, but still present") }

	_ = fmt.Sprintf("%s", mb)
}


func TestFlush(t *testing.T) {
	mb := NewMsgBuffer()

	ch := make(chan []*adsb.CompositeMsg, 3)

	mb.FlushChannel = ch
	mb.MaxMessageAge,mb.MinPublishInterval = 0,0 // Immediate dispatch
	
	messages :=  msgs(maybeAddSBS)
	messages = append(messages, messages[len(messages)-1]) // Let's have two position packets to flush
	for _,msg := range messages {
		mb.Add(&msg)
	}

	if len(ch) != 2 { t.Errorf("channel does not have two items (has %d)", len(ch)) }
}
