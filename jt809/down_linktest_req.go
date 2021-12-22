package jt809

import "fmt"

// 从链路连接保持请求消息
// 链路类型：从链路。
// 消息方向：上级平台往下级平台
// 业务数据类型标识： DOWN_LINKTEST_REQ
// 描述：从链路建立成功后，上级平台向下级平台发送从链路连接保持请求消息，以保持从链路的连接
// 状态。
// 从链路连接保持请求消息，数据体为空。
type DownLinkTestReq struct {
	*headerSeter
}

func NewDownLinkTestReq() *DownLinkTestReq {
	p := &DownLinkTestReq{}
	p.headerSeter = newHeaderSeter(DOWN_LINKTEST_REQ)
	return p
}

func (p *DownLinkTestReq) LinkType() LinkType {
	return DownLinkOnly
}

func (p *DownLinkTestReq) String() string {
	return fmt.Sprintf("DownLinkTestReq{Header:%s", p.Header())
}
