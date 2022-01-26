package jt809

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/lai323/bytecodec"
)

// 主链路车辆动态信息交换业务
// 链路类型：主链路。
// 消息方向：下级平台往上级平台。
// 业务数据类型标识： UP_EXG_MSG.
// 描述：下级平台向上级平台发送车辆动态信息交换业务数据包，其数据体规定见表18。
type UpExgMsg struct {
	*headerSetter
	*subPacketSetter
	VehicleNo    []byte `bytecodec:"length:21"` // 车牌号 21 字节
	VehicleColor byte   // 车牌颜色，按照 JT/T415-2006 中 5.4.12 的规定
	DataType     uint16 // 子业务类型标识
	DataLength   uint32 // 后续数据长度
	// GNSSData     GNSSData // 36 字节
}

func NewUpExgMsg() *UpExgMsg {
	p := &UpExgMsg{}
	p.headerSetter = newHeaderSeter(UP_EXG_MSG)
	p.subPacketSetter = newSubPacketSeter()
	return p
}

func (p UpExgMsg) SubType() uint16 {
	return p.DataType
}

func (p *UpExgMsg) SetSubType(t uint16) {
	p.DataType = t
}

func (p UpExgMsg) SubLength() uint32 {
	return p.DataLength
}

func (p *UpExgMsg) SetSubLength(l uint32) {
	p.DataLength = l
}

func (p UpExgMsg) LinkType() LinkType {
	return UpLink
}

func (p UpExgMsg) String() string {
	return fmt.Sprintf("UpExgMsg{Header:%s, VehicleNo:%s, VehicleColor:%d, DataType:%#04x, DataLength:%d, SubPacket:%s}", p.Header(), p.VehicleNo, p.VehicleColor, p.DataType, p.DataLength, p.SubPacket())
}

func GNSSDataDate(t time.Time) []byte {
	d := byte(t.Day())
	m := byte(t.Month())
	y := uint16(t.Year())
	ybytes := make([]byte, 2)
	binary.BigEndian.PutUint16(ybytes, y)
	return []byte{d, m, ybytes[0], ybytes[1]}
}

func GNSSDataTime(t time.Time) []byte {
	h := byte(t.Hour())
	m := byte(t.Minute())
	s := byte(t.Second())
	return []byte{h, m, s}
}

// 上传车辆注册信息消息
// 子业务类型标识： UP_EXG_MSG_REAL_LOCATION
// 描述：主要描述车辆的实时定位信息，本条消息服务端无需应答。
type UpExgMsgRealLocation struct {
	Encrypt   byte            // 该字段标识传输的定位信息是否使用国家测绘局批准的地图保密插件进行加密。加密标识：1-已加密，0-未加密
	Date      []byte          `bytecodec:"length:4"` // 日月年(dmy), 4 字节 年的表示是先将年转换成两位十六进制数，2009表示为 0x07 0xD9
	Time      []byte          `bytecodec:"length:3"` // 时分秒(hms) 3 字节
	Lon       uint32          // 经度，单位为1e-06度
	Lat       uint32          // 纬度，单位为1e-06度
	Vec1      uint16          // 速度，指卫星定位车载终端设备上传的行车速度信息，为必填项单位为千米每小时(km/h)
	Vec2      uint16          // 行驶记录速度，指车辆行驶记录设备上传的行车速度信息，单位为千米每小时(km/h)
	Vec3      uint32          // 车辆当前总里程数，指车辆上传的行车里程数，单位为千米(km)
	Direction uint16          // 方向，0~359,单位为度(°),正北为0,顺时针
	Altitude  uint16          // 海拔高度，单位为米(m)
	State     *LocationStatus // 车辆状态，二进制表示：B31B30…B2B1B0。具体定义按照JT/T808--2011中表17的规定
	Alarm     *LocationAlarm  // 报警状态，二进制表示，0表示正常，1表示报警：B31B30B29........B2B1B0。具体定义按照JT/T808-2011中表18的规定
}

func NewUpExgMsgRealLocation() *UpExgMsgRealLocation {
	return &UpExgMsgRealLocation{
		State: &LocationStatus{},
		Alarm: &LocationAlarm{},
	}
}

func (p UpExgMsgRealLocation) SubType() uint16 {
	return UP_EXG_MSG_REAL_LOCATION
}

func (p UpExgMsgRealLocation) String() string {
	return fmt.Sprintf("UpExgMsgRealLocation{Encrypt:%d, Date:%#x, Time:%#x, Lon:%d, Lat:%d, Vec1:%d, Vec2:%d, Vec3:%d, Direction:%d, Altitude:%d, State:%s, Alarm:%s}", p.Encrypt, p.Date, p.Time, p.Lon, p.Lat, p.Vec1, p.Vec2, p.Vec3, p.Direction, p.Altitude, p.State, p.Alarm)
}

type LocationStatus struct {
	ACC           bool // 0    0:ACC关；1:ACC开
	Location      bool // 1    0:未定位；1:定位
	LatitudeSouth bool // 2    0:北纬；1:南纬
	LongitudeWest bool // 3    0:东经；1:西经
	OnOperate     bool // 4    0:运营状态；1:停运状态
	Encrypted     bool // 5    0:经纬度未经保密插件加密；1:经纬度已经保密插件加密
	// 6-9 保留
	Gas        bool // 10   0:车辆油路正常；1:车辆油路断开
	Circuit    bool // 11   0:车辆电路正常：1:车辆电路断开
	DoorLocked bool // 12   0:车门解锁；1:车门加锁
	// 13-31 保留
}

func (s LocationStatus) String() string {
	return fmt.Sprintf("LocationStatus{Value:%#032b}", s.Uint32())
}

func (s LocationStatus) Uint32() uint32 {
	bools := []bool{}
	bools = append(bools, s.ACC)
	bools = append(bools, s.Location)
	bools = append(bools, s.LatitudeSouth)
	bools = append(bools, s.LongitudeWest)
	bools = append(bools, s.OnOperate)
	bools = append(bools, s.Encrypted)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, s.Gas)
	bools = append(bools, s.Circuit)
	bools = append(bools, s.DoorLocked)
	return uint32(BoolSliceToBin(bools))
}

func (s *LocationStatus) MarshalBytes(cs *bytecodec.CodecState) error {
	v := s.Uint32()
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	cs.Write(b)
	return nil
}

func (s *LocationStatus) UnmarshalBytes(cs *bytecodec.CodecState) error {
	b := make([]byte, 4)
	cs.ReadFull(b)
	v := binary.BigEndian.Uint32(b)
	bools := BinToBoolSlice(uint64(v))
	s.ACC = bools[0]
	s.Location = bools[1]
	s.LatitudeSouth = bools[2]
	s.LongitudeWest = bools[3]
	s.OnOperate = bools[4]
	s.Encrypted = bools[5]
	s.Gas = bools[10]
	s.Circuit = bools[11]
	s.DoorLocked = bools[12]
	return nil
}

type LocationAlarm struct {
	Emergency         bool // 0  1:紧急报警，触动报警开关后触发     收到应答后清零
	Speeding          bool // 1  1;超速报警                      标志维持至报警条件解除
	Fatigue           bool // 2  1:疲劳驾驶报警                   标志维持至报警条件解除
	DangerousBehavior bool // 3  1:预警                        收到应答后清零
	GNSSError         bool // 4  1:GNSS 模块发生故障报警          标志维持至报警条件解除
	GNSSAntennaError  bool // 5  1:GNSS 天线未接或被剪断报警       标志维持至报警条件解除
	GNSSShortCircuit  bool // 6  1:GNSS 天线短路报警             标志维持至报警条件解除
	PowerUndervoltage bool // 7  1:终端主电源欠压报警             标志维持至报警条件解除
	PowerDown         bool // 8  1:终端主电源掉电报警             标志维持至报警条件解除
	LCDError          bool // 9  1:终端 LCD 或显示器故障报警       标志维持至报警条件解除
	TTSError          bool // 10 1:TTS 模块故障报警               标志维持至报警条件解除
	CameraErr         bool // 11 1:摄像头故障报警                标志维持至报警条件解除
	// 12-17 保留
	FatigueDaily      bool // 18 1:当天累计驾驶超时报警           标志维持至报警条件解除
	StopTimeout       bool // 19 1:超时停车报警                  标志维持至报警条件解除
	RangeArea         bool // 20 1:进出区域报警                  收到应答后清零
	RangeRoute        bool // 21 1:进出路线报警                  收到应答后清零
	DriveTimeError    bool // 22 1:路段行驶时间不足/过长报警      收到应答后清零
	RouteDeviate      bool // 23 1:路线偏离报警                 标志维持至报警条件解除
	VSSError          bool // 24 1:车辆 VSS 故障                标志持至报警条件解除
	GasQuantityError  bool // 25 1:车辆油量异常报警              标志维持至报警条件解除
	Stolen            bool // 26 1:车辆被盗报警（通过车辆防盗器）    标志维持至报警条件解除
	IllegalStart      bool // 27 1:车辆非法点火报警              收到应答后清零
	IllegalMove       bool // 28 1:车辆非法位移报警              收到应答后清零
	CollisionRollover bool // 29 1:碰撞侧翻报警                   标志维持至报警条件解除
	// 30-31 保留
}

func (s LocationAlarm) String() string {
	return fmt.Sprintf("LocationAlarm{Value:%#032b}", s.Uint32())
}

func (s LocationAlarm) Uint32() uint32 {
	bools := []bool{}
	bools = append(bools, s.Emergency)
	bools = append(bools, s.Speeding)
	bools = append(bools, s.Fatigue)
	bools = append(bools, s.DangerousBehavior)
	bools = append(bools, s.GNSSError)
	bools = append(bools, s.GNSSAntennaError)
	bools = append(bools, s.GNSSShortCircuit)
	bools = append(bools, s.PowerUndervoltage)
	bools = append(bools, s.PowerDown)
	bools = append(bools, s.LCDError)
	bools = append(bools, s.TTSError)
	bools = append(bools, s.CameraErr)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, false)
	bools = append(bools, s.FatigueDaily)
	bools = append(bools, s.StopTimeout)
	bools = append(bools, s.RangeArea)
	bools = append(bools, s.RangeRoute)
	bools = append(bools, s.DriveTimeError)
	bools = append(bools, s.RouteDeviate)
	bools = append(bools, s.VSSError)
	bools = append(bools, s.GasQuantityError)
	bools = append(bools, s.Stolen)
	bools = append(bools, s.IllegalStart)
	bools = append(bools, s.IllegalMove)
	bools = append(bools, s.CollisionRollover)
	return uint32(BoolSliceToBin(bools))
}

func (s *LocationAlarm) MarshalBytes(cs *bytecodec.CodecState) error {
	v := s.Uint32()
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	cs.Write(b)
	return nil
}

func (s *LocationAlarm) UnmarshalBytes(cs *bytecodec.CodecState) error {
	b := make([]byte, 4)
	cs.ReadFull(b)
	v := binary.BigEndian.Uint32(b)
	bools := BinToBoolSlice(uint64(v))
	s.Emergency = bools[0]
	s.Speeding = bools[1]
	s.Fatigue = bools[2]
	s.DangerousBehavior = bools[3]
	s.GNSSError = bools[4]
	s.GNSSAntennaError = bools[5]
	s.GNSSShortCircuit = bools[6]
	s.PowerUndervoltage = bools[7]
	s.PowerDown = bools[8]
	s.LCDError = bools[9]
	s.TTSError = bools[10]
	s.CameraErr = bools[11]
	s.FatigueDaily = bools[18]
	s.StopTimeout = bools[19]
	s.RangeArea = bools[20]
	s.RangeRoute = bools[21]
	s.DriveTimeError = bools[22]
	s.RouteDeviate = bools[23]
	s.VSSError = bools[24]
	s.GasQuantityError = bools[25]
	s.Stolen = bools[26]
	s.IllegalStart = bools[27]
	s.IllegalMove = bools[28]
	s.CollisionRollover = bools[29]
	return nil
}
