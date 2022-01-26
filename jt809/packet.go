package jt809

import (
	"fmt"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// 链路管理类
const (
	UP_CONNECT_REQ         uint16 = 0x1001 // 主链路登录请求消息 主链路
	UP_CONNECT_RSP         uint16 = 0x1002 // 主链路登录应答消息 主链路
	UP_DISCONNECT_REQ      uint16 = 0x1003 // 主链路注销请求消息 主链路
	UP_DISCONNECT_RSP      uint16 = 0x1004 // 主链路注销应答消息 主链路
	UP_LINKTEST_REQ        uint16 = 0x1005 // 主链路连接保持请求消息 主链路
	UP_LINKTEST_RSP        uint16 = 0x1006 // 主链路连接保持应答消息 主链路
	UP_DISCONNECT_INFORM   uint16 = 0x1007 // 主链路断开通知消息 从链路
	UP_CLOSELINK_INFORM    uint16 = 0x1008 // 下级平台主动关闭链路通知消息 从链路
	DOWN_CONNECT_REQ       uint16 = 0x9001 // 从链路连接请求消息 从链路
	DOWN_CONNECT_RSP       uint16 = 0x9002 // 从链路连接应答消息 从链路
	DOWN_DISCONNECT_REQ    uint16 = 0x9003 // 从链路注销请求消息 从链路
	DOWN_DISCONNECT_RSP    uint16 = 0x9004 // 从链路注销应答消息 从链路
	DOWN_LINKTEST_REQ      uint16 = 0x9005 // 从链路连接保持请求消息 从链路
	DOWN_LINKTEST_RSP      uint16 = 0x9006 // 从链路连接保持应答消息 从链路
	DOWN_DISCONNECT_INFORM uint16 = 0x9007 // 从链路断开通知消息 主链路
	DOWN_CLOSELINK_INFORM  uint16 = 0x9008 // 上级平台主动关闭链路通知消息 主链路
)

// 信息统计类
const DOWN_TOTAL_RECV_BACK_MSG uint16 = 0x9101 // 接收定位信息数量通知消息 从链路

//车辆动态信息交换类
const (
	UP_EXG_MSG               uint16 = 0x1200 // 主链路动态信息交换消息 主链路
	UP_EXG_MSG_REAL_LOCATION uint16 = 0x1202 // 实时上传车辆定位信息 主链路
)

// 车牌颜色，按照 JT/T415-2006 中 5.4.12 的规定
const (
	PlateColorBlue   = 1
	PlateColorYellow = 2
	PlateColorBlack  = 3
	PlateColorWhite  = 4
	PlateColorOther  = 9 // 其他
)

// SerialNo 占用四个字节，为发送信息的序列号，用于接收方检测是否有信息的丢失。上级平台和下级平台按自己发送数据包的个数计数，互不影响。程序开始运行时等于零，发送第一帧数据时开始计数，到最大数后自动归零
// Encrypt 用来区分报文是否进行加密，如果标识为1,则说明对后续相应业务的数据体采用 EncryptKey 对应的密钥进行加密处理。如果标识为0,则说明不进行加密处理
type Header struct {
	Length       uint32 // 数据长度(包括头标识、数据头、数据体和尾标识)
	SerialNo     uint32 // 报文序列号
	Type         uint16 // 业务数据类型
	GNSSCenterID uint32 // 下级平台接入码，上级平台给下级平台分配的唯一标识号
	Version      []byte `bytecodec:"length:3"` // 协议版本号标识， 3 字节 上下级平台之间采用的标准协议版本编号；长度为三个字节来表示： 0x01 0x02 x0F 表示的版本号是V1.2.15,依此类推
	Encrypt      byte   // 报文加密标识位：0表示报文不加密，1表示报文加密
	EncryptKey   uint32 // 数据加密的密钥，长度为四个字节
}

func (h Header) String() string {
	return fmt.Sprintf("Header{Length:%d SerialNo:%d, Type:%#04x, GNSSCenterID:%d, Version:%v, Encrypt:%d, EncryptKey:%d}", h.Length, h.SerialNo, h.Type, h.GNSSCenterID, h.Version, h.Encrypt, h.EncryptKey)
}

// 数据头是定长的
const HeaderLength = 4 + 4 + 2 + 4 + 3 + 1 + 4 // 22

type LinkType byte

const (
	UpLink       LinkType = 1
	UpLinkOnly   LinkType = 2
	DownLink     LinkType = 3
	DownLinkOnly LinkType = 4
)

type SubPacket interface {
	String() string
	SubType() uint16
}

type SubPacketSetter interface {
	SubType() uint16
	SetSubType(uint16)
	SubLength() uint32
	SetSubLength(uint32)
	SubPacket() SubPacket
	SetSubPacket(subpacket SubPacket)
}

type Packet interface {
	String() string
	LinkType() LinkType
	Header() *Header
	SetHeader(Header)
}

type headerSetter struct {
	header Header
}

func newHeaderSeter(t uint16) *headerSetter {
	return &headerSetter{
		header: Header{
			Type:    t,
			Version: []byte{1, 0, 0},
		},
	}
}

func (seter *headerSetter) Header() *Header {
	return &seter.header
}

func (seter *headerSetter) SetHeader(h Header) {
	seter.header = h
}

type subPacketSetter struct {
	subpacket SubPacket
}

func newSubPacketSeter() *subPacketSetter {
	return &subPacketSetter{}
}

func (seter *subPacketSetter) SubPacket() SubPacket {
	return seter.subpacket
}

func (seter *subPacketSetter) SetSubPacket(subpacket SubPacket) {
	seter.subpacket = subpacket
}

var newPacketMap = map[uint16]func() Packet{
	UP_CONNECT_REQ:    func() Packet { return NewUpConnectReq() },
	UP_CONNECT_RSP:    func() Packet { return NewUpConnectRsp() },
	UP_LINKTEST_REQ:   func() Packet { return NewUpLinkTestReq() },
	UP_LINKTEST_RSP:   func() Packet { return NewUpLinkTestRsp() },
	DOWN_CONNECT_REQ:  func() Packet { return NewDownConnectReq() },
	DOWN_CONNECT_RSP:  func() Packet { return NewDownConnectRsp() },
	DOWN_LINKTEST_REQ: func() Packet { return NewDownLinkTestReq() },
	DOWN_LINKTEST_RSP: func() Packet { return NewDownLinkTestRsp() },
	UP_EXG_MSG:        func() Packet { return NewUpExgMsg() },
}

var newSubPacketMap = map[uint16]func() SubPacket{
	UP_EXG_MSG_REAL_LOCATION: func() SubPacket { return NewUpExgMsgRealLocation() },
}

func FixedLengthString(s string, length int, gbk bool) []byte {
	b := make([]byte, length)
	if !gbk {
		copy(b, []byte(s))
		return b
	}
	gbkb, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if err != nil {
		panic(err)
	}
	copy(b, gbkb)
	return b
}

type SerialNoGenerater struct {
	sn    uint32
	snMap map[uint16]uint32
	mtx   sync.Mutex
}

func NewSerialNoGenerater() *SerialNoGenerater {
	return &SerialNoGenerater{snMap: map[uint16]uint32{}}
}

// 按照消息的子业务数据类型分别编号
func (g *SerialNoGenerater) GetByType(packetType uint16) uint32 {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	sn := g.snMap[packetType]
	g.snMap[packetType] += 1
	return sn
}

func (g *SerialNoGenerater) Get() uint32 {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	sn := g.sn
	g.sn += 1
	return sn
}
