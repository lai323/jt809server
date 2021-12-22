package jt809

import "fmt"

// 主链路连接保持应答消息
// 链路类型：主链路
// 消息方向：上级平台往下级平台。
// 业务数据类型标识：UP_LINKTEST_RSP.
// 描述：上级平台收到下级平台的主链路连接保持请求消息后，向下级平台返回主链路连接保持应答消息，保持主链路的连接状态。
// 主链路连接保持应答消息，数据体为空。
type UpLinkTestRsp struct {
	*headerSeter
}

func NewUpLinkTestRsp() *UpLinkTestRsp {
	p := &UpLinkTestRsp{}
	p.headerSeter = newHeaderSeter(UP_LINKTEST_RSP)
	return p
}

func (p *UpLinkTestRsp) LinkType() LinkType {
	return UpLinkOnly
}

func (p *UpLinkTestRsp) String() string {
	return fmt.Sprintf("UpLinkTestRsp{Header:%s", p.Header())
}
