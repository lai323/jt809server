package jt809

import "fmt"

// 主链路连接保持请求消息
// 链路类型：主链路
// 消息方向：下级平台往上级平台。
// 业务数据类型标识： UP_LINKTEST_REQ.
// 描述：下级平台向上级平台发送主链路连接保持请求消息，以保持主链路的连接。
// 主链路连接保持请求消息，数据体为空。
type UpLinkTestReq struct {
	*headerSetter
}

func NewUpLinkTestReq() *UpLinkTestReq {
	p := &UpLinkTestReq{}
	p.headerSetter = newHeaderSeter(UP_LINKTEST_REQ)
	return p
}

func (p UpLinkTestReq) LinkType() LinkType {
	return UpLinkOnly
}

func (p UpLinkTestReq) String() string {
	return fmt.Sprintf("UpLinkTestReq{Header:%s", p.Header())
}
