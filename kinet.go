package kinet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
  "math"
  "encoding/hex"
  "image/color"
  "io/ioutil"
  "log"
  "strings"
)

type header struct {
	Magic    uint32
	Version  uint16
	Type     uint16
	SeqNum  uint32
}

const (
  UINT8_MAX = 1<<8 - 1
  UINT16_MAX = 1<<16 - 1
)

const (
  KINET_MAGIC = 0x4adc0104
  KINET_VERSION = 0x0001
  KINET_NUM_PIXELS = UINT8_MAX / 3
)

// kinet packet types
const (
  KINET_SUP_REQ = 0x0001
  KINET_SUP_RESP = 0x0002
  KINET_SET_IP = 0x0003
  KINET_SET_UNIVERSE = 0x0005
  KINET_SET_NAME = 0x0006
  KINET_PORT_REQ = 0x000a
  KINET_PORT_RESP = 0x000a
  KINET_SET_COLORS = 0x0101
  KINET_FIXTURE_REQ = 0x0201
  KINET_FIXTURE_RESP = 0x0202
  KINET_CHAN_REQ = 0x0203
  KINET_CHAN_RESP = 0x0204
)

type kinet_set_colors struct {
	header
	Port    uint8
	Padding uint8
	Flags   uint16
	Timer   uint32
	Uni     uint8
  //Colors  []byte 
}

type kinet_sup_req struct {
	header
	Port    uint8
	Padding uint8
	Flags   uint16
}

type kinet_sup_resp struct {
	header
	IP [4]byte
	Mac [6]byte
	Version uint16
	Serial uint32
	Universe uint32
}

type kinet_fix_req struct {
	header
	Padding uint32
}

type kinet_fix_resp struct {
	header
	Serial uint32
}

type kinet_chan_req struct {
	header
  Serial uint32
  Magic uint32
}

type kinet_chan_resp struct {
	header
  Serial uint32
  Magic uint16
  Channel uint8
  OK uint8
}

type PowerSupply struct {
  Name string
  IP string
  Mac string
	ProtocolVersion string
	Serial string
	Universe string
  Manufacturer string
  Type string
  FWVersion string
  Fixtures []*Fixture
}

type Fixture struct {
	Serial string
  Channel uint8
  Color color.Color
  PS *PowerSupply `json:"-"`
}

func init() {
  log.SetOutput(ioutil.Discard)
}

func gammaCorrect(c color.Color) color.Color {
  r, g, b, _ := c.RGBA()
  r_fix := uint8(math.Pow(float64(r) / UINT16_MAX, 2.2) * 255.0)
  g_fix := uint8(math.Pow(float64(g) / UINT16_MAX, 2.2) * 255.0)
  b_fix := uint8(math.Pow(float64(b) / UINT16_MAX, 2.2) * 255.0)
  return color.RGBA{r_fix, g_fix, b_fix, 0xff}
}

func (fixture *Fixture) SendColor(c color.Color) {
  colors := make([]byte, UINT8_MAX)

  // FIXME save last byte array?
  for i := range fixture.PS.Fixtures {
    f := fixture.PS.Fixtures[i]
    r, g, b, _ := f.Color.RGBA()
    colors[f.Channel] = byte((r * UINT8_MAX) / UINT16_MAX)
    colors[f.Channel + 1] = byte((g * UINT8_MAX) / UINT16_MAX)
    colors[f.Channel + 2] = byte((b * UINT8_MAX) / UINT16_MAX)
  }

  c = gammaCorrect(c)
  r, g, b, _ := c.RGBA()
  // Go color interface returns 32 bit ints, but max is of a 16 bit int
  // We need them scaled such that they are 8 bit
  colors[fixture.Channel] = byte((r * UINT8_MAX) / UINT16_MAX)
  colors[fixture.Channel + 1] = byte((g * UINT8_MAX) / UINT16_MAX)
  colors[fixture.Channel + 2] = byte((b * UINT8_MAX) / UINT16_MAX)

  fixture.Color = c

	d := kinet_set_colors{
    header{KINET_MAGIC, KINET_VERSION, KINET_SET_COLORS, 0x00000000},
		0x00, 0x00, 0x0000, 0xffffffff, 0x00,
  }

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)

  sendKinetPacket(fixture.PS.IP, append(buf.Bytes(), colors...), nil)
}

func (ps *PowerSupply) SendColors(c []color.Color) {
  //if len(c) > (KINET_NUM_PIXELS) {
    // return error
  //}

  colors := make([]byte, len(c) * 3)
  for i := 0; i < len(c); i++ {
    r, g, b, _ := c[i].RGBA()
    // Go color interface returns 32 bit ints, but max is of a 16 bit int
    // We need them scaled such that they are 8 bit
    colors[(i*3)] = byte((r * UINT8_MAX) / UINT16_MAX)
    colors[(i*3)+1] = byte((g * UINT8_MAX) / UINT16_MAX)
    colors[(i*3)+2] = byte((b * UINT8_MAX) / UINT16_MAX)
  }

  if ps.Fixtures != nil {
    for i := range ps.Fixtures {
      ps.Fixtures[i].Color = c[ps.Fixtures[i].Channel / 3]
    }
  }

	d := kinet_set_colors{
    header{KINET_MAGIC, KINET_VERSION, KINET_SET_COLORS, 0x00000000},
		0x00, 0x00, 0x0000, 0xffffffff, 0x00,
  }

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)


  sendKinetPacket(ps.IP, append(buf.Bytes(), colors...), nil)
}

func (fixture *Fixture) DiscoverChannel() {
  serial, _ := hex.DecodeString(fixture.Serial)
  ss := bytes.NewReader(serial)
  var s uint32
  binary.Read(ss, binary.BigEndian, &s)

	d := kinet_chan_req{
    header{KINET_MAGIC, KINET_VERSION, KINET_CHAN_REQ, 0x00000000},
		s, 0x96d60041,
  }

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)

  resp := make(chan []byte)
  go sendKinetPacket(fixture.PS.IP, buf.Bytes(), resp)

  rbuf := <-resp
  var c kinet_chan_resp
  read := bytes.NewReader(rbuf)
  binary.Read(read, binary.LittleEndian, &c)
  fixture.Channel = c.Channel
}

func (ps *PowerSupply) DiscoverFixtures() {
	d := kinet_fix_req{
    header{KINET_MAGIC, KINET_VERSION, KINET_FIXTURE_REQ, 0x00000000},
		0x00000000,
  }
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)

  resp := make(chan []byte)
  go sendKinetPacket(ps.IP, buf.Bytes(), resp)

	fixtures := []*Fixture{}
  for rbuf := range resp {
		var resp kinet_fix_resp
    read := bytes.NewReader(rbuf)
    binary.Read(read, binary.LittleEndian, &resp)
    fixture := Fixture{fmt.Sprintf("%X", resp.Serial), 0, color.Black, ps}
    fixtures = append(fixtures, &fixture)
  }

  for i := range fixtures {
    fixtures[i].DiscoverChannel()
  }
  ps.Fixtures = fixtures
}

func DiscoverSupplies() []*PowerSupply {
	d := kinet_sup_req{
    header{KINET_MAGIC, KINET_VERSION, KINET_SUP_REQ, 0x00000000},
		0x0a,
    0x87,
    0x8988,
  }
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)

  resp := make(chan []byte)
  go sendKinetPacket("255.255.255.255", buf.Bytes(), resp)

	power_supplies := []*PowerSupply{}
  for rbuf := range resp {
		var resp kinet_sup_resp
    read := bytes.NewReader(rbuf)
    binary.Read(read, binary.LittleEndian, &resp)

    ps := PowerSupply{}
    ids := []string{}

    // Skip first 2 bytes (M:)
    var start int = 0
    rbuf = rbuf[34:]
		for i := range rbuf {
      if rbuf[i] == ':' {
        ids = append(ids, string(rbuf[start:i-2]))
        start = i+1
      } else if rbuf[i] == 0x00 {
         ps.Name = string(rbuf[i+1:])
         ps.Name = ps.Name[:strings.Index(ps.Name,"\x00")]
        break
      }
		}
    ps.Manufacturer = ids[0]
    ps.Type = ids[1]
    ps.FWVersion = ids[2]

    // FIXME
    ps.ProtocolVersion = fmt.Sprintf("%v", resp.Version)
    ps.Serial = fmt.Sprintf("%x", resp.Serial)
    ps.Universe = fmt.Sprintf("%v", resp.Universe)

    ps.IP = strings.Replace(fmt.Sprintf("%v", resp.IP), " ", ".", -1)
    ps.IP = ps.IP[1:len(ps.IP)-1]

    ps.Mac = strings.Replace(fmt.Sprintf("% x", resp.Mac), " ", ":", -1)

    power_supplies = append(power_supplies, &ps)
	}
	return power_supplies
}

func Discover() []*PowerSupply {
  power_supplies := DiscoverSupplies()
  for i := range power_supplies {
    power_supplies[i].DiscoverFixtures()
  }
	return power_supplies
}

// FIXME
func sendKinetPacket(host string, packet []byte, resp chan []byte) {
	laddr, _ := net.ResolveUDPAddr("udp", ":0")
	raddr, _ := net.ResolveUDPAddr("udp", host + ":6038")
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		os.Exit(1)
	}
  // need to modify this, for no need to wait for fixture lookup or color set 
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

  log.Printf("% X", packet)
	_, err = conn.WriteToUDP(packet, raddr)
	if err != nil {
		os.Exit(2)
	}

  if resp == nil {
    return
  }

	var n int
	for {
		rbuf := make([]byte, 200)
		n, _, err = conn.ReadFromUDP(rbuf)
		if err != nil || n == 0 {
			conn.Close()
      close(resp)
      return
		}
    resp <- rbuf[:n]
  }
}
