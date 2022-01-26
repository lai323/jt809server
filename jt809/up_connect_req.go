package jt809

import "fmt"

// 主链路登录请求消息
// 链路类型：主链路。
// 消息方向：下级平台往上级平台。
// 业务数据类型标识：UP_ CONNECT_REQ.
// 描述：下级平台向上级平台发送用户名和密码等登录信息。
type UpConnectReq struct {
	*headerSetter
	UserID       uint32 // 用户名
	Password     []byte `bytecodec:"length:8"`  // 密码 8 字节
	DownLinkIP   []byte `bytecodec:"length:32"` // 下级平台提供对应的从链路服务端 IP 地址 32 字节
	DownLinkPort uint16 // 下级平台提供对应的从链路服务端口号
}

func NewUpConnectReq() *UpConnectReq {
	p := &UpConnectReq{}
	p.headerSetter = newHeaderSeter(UP_CONNECT_REQ)
	return p
}

func (p UpConnectReq) LinkType() LinkType {
	return UpLinkOnly
}

func (p UpConnectReq) String() string {
	return fmt.Sprintf("UpConnectReq{Header:%s UserID:%d, Password:%s, DownLinkIP:%s, DownLinkPort:%d}", p.Header(), p.UserID, p.Password, p.DownLinkIP, p.DownLinkPort)
}
