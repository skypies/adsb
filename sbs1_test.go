// go test -v github.com/skypies/adsb
package adsb

import(
	"fmt"
	"bufio"
	"strings"
	"testing"
)

var(
	sbs = `
MSG,7,1,1,A81BD0,1,2015/11/27,21:31:02.722,2015/11/27,21:31:02.721,,20150,,,,,,,,,,0
MSG,3,1,1,A81BD0,1,2015/11/27,21:31:03.354,2015/11/27,21:31:03.316,,20125,,,36.69804,-121.86007,,,,,,0
MSG,3,1,1,A81BD0,1,2015/11/27,21:31:03.704,2015/11/27,21:31:03.716,,20125,,,36.69830,-121.86017,,,,,,0
`
	extsbs = `
MLAT,3,1,1,A76E37,1,2016/03/10,18:22:22.989,2016/03/10,18:22:22.989,,28211,497,66,36.8347,-120.4883,1696,,,,,,,,
MLAT,3,1,1,A2C635,1,2016/03/10,18:22:23.220,2016/03/10,18:22:23.220,,37559,379,316,36.3631,-120.3861,-884,,,,,,,,
MLAT,3,1,1,A24757,1,2016/03/10,18:22:23.527,2016/03/10,18:22:23.527,,39006,465,150,35.9798,-119.7215,-5,,,,,,,,
MLAT,3,1,1,A81A3E,1,2016/03/10,18:22:24.115,2016/03/10,18:22:24.115,,21113,399,143,36.8268,-121.4215,1003,,,,,,,,
MLAT,3,1,1,A7BBE9,1,2016/03/10,18:22:24.180,2016/03/10,18:22:24.180,,8901,217,296,37.1378,-122.6959,3,,,,,,,,
MLAT,3,1,1,AB5024,1,2016/03/10,18:22:24.183,2016/03/10,18:22:24.183,,6628,238,321,37.0451,-121.7235,-818,,,,,,,,
`
	maskedsbs = `
MLAT,3,1,1,~A76E37,1,2016/03/10,18:22:22.989,2016/03/10,18:22:22.989,,28211,497,66,36.8347,-120.4883,1696,,,,,,,,
`
)

func TestSBSParsing(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(sbs))
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" { continue } // blank lines
		m := Msg{}
		fmt.Printf(" --- %s ---\n", text)
		if err := m.FromSBS1(text); err != nil {
			t.Errorf("parse fail on '%s': %v", text, err)
		}
		if m.IsMLAT() {
			t.Errorf("regular parse is MLAT !")
		}
		if m.IsMasked() {
			t.Errorf("regular parse is Masked !")
		}
	}
}

func TestExtendedSBSParsing(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(extsbs))
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" { continue } // blank lines
		fmt.Printf(" --- %s ---\n", text)
		m := Msg{}
		if err := m.FromSBS1(text); err != nil {
			t.Errorf("parse fail on '%s': %v", text, err)
		}
		if ! m.IsMLAT() {
			t.Errorf("extended parse not MLAT !")
		}
		if ! m.HasPosition() {
			t.Errorf("extended parse has no position !\n%s\n%s", text, m.ToSBS1())
		}
	}
}

func TestMaskededSBSParsing(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(maskedsbs))
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" { continue } // blank lines
		fmt.Printf(" --- %s ---\n", text)
		m := Msg{}
		if err := m.FromSBS1(text); err != nil {
			t.Errorf("parse fail on '%s': %v", text, err)
		}
		if ! m.IsMasked() {
			t.Errorf("not Masked !")
		}
	}
}
