package jt809

import "fmt"

// 从链路连接保持应答消息
// 链路类型：从链路。
// 消息方向：下级平台往上级平台。
// 业务数据类型标识： DOWN_LINKTEST_RSP.
// 描述：下级平台收到上级平台链路连接保持请求消息后，向上级平台返回从链路连接保持应答消息，保持从链路连接状态
// 从链路连接保持应答消息，数据体为空。
type DownLinkTestRsp struct {
	*headerSeter
}

func NewDownLinkTestRsp() *DownLinkTestRsp {
	p := &DownLinkTestRsp{}
	p.headerSeter = newHeaderSeter(DOWN_LINKTEST_RSP)
	return p
}

func (p *DownLinkTestRsp) LinkType() LinkType {
	return DownLinkOnly
}

func (p *DownLinkTestRsp) String() string {
	return fmt.Sprintf("DownLinkTestRsp{Header:%s", p.Header())
}
