package jt809

import "fmt"

// 从链路连接请求消息
// 链路类型：从链路。
// 消息方向：上级平台往下级平台。
// 业务数据类型标识： DOWN_CONNECT_REQ.
// 描述：主链路建立连接后，上级平台向下级平台发送从链路连接请求消息，以建立从链路连接
type DownConnectReq struct {
	*headerSeter
	VerifyCode uint32 // 主链路登录应答的校验码
}

func NewDownConnectReq() *DownConnectReq {
	p := &DownConnectReq{}
	p.headerSeter = newHeaderSeter(DOWN_CONNECT_REQ)
	return p
}

func (p *DownConnectReq) LinkType() LinkType {
	return DownLinkOnly
}

func (p *DownConnectReq) String() string {
	return fmt.Sprintf("DownConnectReq{Header:%s VerifyCode:%d}", p.Header(), p.VerifyCode)
}
