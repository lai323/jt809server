package jt809

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lai323/bytecodec"
)

func TestUpConnectReq(t *testing.T) {
	pktbytes := mustHexDecodeString("5B000000480000008510010133EFB8010000010003E8B2D37D9CC4900C77DC78F8676527D8AE12243CFB64CC2FBA619AEFAD33ACCB3256F67BFF19DF33097841098665703FE36E5D")
	p := NewUpConnectReq()
	h := p.Header()
	h.SerialNo = 133
	h.Encrypt = 1
	h.EncryptKey = 256178
	h.GNSSCenterID = 20180920
	p.UserID = 20180920
	p.Password = FixedLengthString("20180920", 8, false)
	p.DownLinkIP = FixedLengthString("127.0.0.1", 32, false)
	p.DownLinkPort = 809

	subtest := map[Packet][]byte{
		p: pktbytes,
	}
	testpacket(t, subtest)
}

type locationAlarmStatus struct {
	DataLength uint16 `bytecodec:"lengthref:Data"`
	Data       locationAlarmStatusData
}

func (s locationAlarmStatus) String() string {
	return fmt.Sprintf("locationAlarmStatus{DataLength:%d, Data:%s}", s.DataLength, s.Data)
}

type locationAlarmStatusData struct {
	Alarm *LocationAlarm
	State *LocationStatus
}

func (s locationAlarmStatusData) String() string {
	return fmt.Sprintf("locationAlarmStatusData{Alarm:%s, State:%s}", s.Alarm, s.State)
}

func TestLocationAlarmStatus(t *testing.T) {
	las := locationAlarmStatus{
		DataLength: 8,
		Data: locationAlarmStatusData{
			Alarm: &LocationAlarm{
				Emergency:         true,
				Speeding:          true,
				CollisionRollover: true,
			},
			State: &LocationStatus{
				ACC:        true,
				Location:   true,
				DoorLocked: true,
			},
		},
	}

	lasBytes := mustHexDecodeString("00082000000300001003")
	bytesRet, err := bytecodec.Marshal(las)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(bytesRet, lasBytes) {
		t.Error("Marshal error", las, strings.ToUpper(hex.EncodeToString(lasBytes)), strings.ToUpper(hex.EncodeToString(bytesRet)))
	}

	lasOut := locationAlarmStatus{}
	err = bytecodec.Unmarshal(lasBytes, &lasOut)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(las, lasOut) {
		t.Error("Unmarshal error", strings.ToUpper(hex.EncodeToString(lasBytes)), las, lasOut)
	}
}

func TestUpExgMsgRealLocation(t *testing.T) {
	pktbytes := mustHexDecodeString("5b0000005a020000008512000133efb8010000010003e8b234fbf83d930e75d07dc1555516ea993c1412cb4afd2da8639aefad17acdf3e511377ce10df3309034109861e736de119cebffef3af0e8a6f9479a1ac36fc416960bd5d")

	gnsstime, _ := time.Parse("2006-01-02 15:04:05", "2021-12-20 12:49:09")
	p := NewUpExgMsg()

	h := p.Header()
	h.SerialNo = 133
	h.Encrypt = 1
	h.EncryptKey = 256178
	h.GNSSCenterID = 20180920

	p.VehicleNo = FixedLengthString("测A12345", 21, true)
	p.VehicleColor = PlateColorYellow
	// p.DataType
	// p.DataLength

	loc := NewUpExgMsgRealLocation()
	loc.Encrypt = 0
	loc.Date = GNSSDataDate(gnsstime)
	loc.Time = GNSSDataTime(gnsstime)
	loc.Lon = 123
	loc.Lat = 123
	loc.Vec1 = 123
	loc.Vec2 = 123
	loc.Vec3 = 123
	loc.Direction = 123
	loc.Altitude = 123
	loc.State = &LocationStatus{ACC: true, Location: true}
	loc.Alarm = &LocationAlarm{}
	p.SetSubPacket(loc)

	// b := mustMarshal(p)
	// t.Log(hex.EncodeToString(b))
	subtest := map[Packet][]byte{
		p: pktbytes,
	}
	testpacket(t, subtest)
}
