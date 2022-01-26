package jt809

import "fmt"

// 从链路连接应答信息
// 链路类型：从链路。
// 消息方向：下级平台往上级平台。
// 业务数据类型标识： DOWN_CONNECT_RSP.
// 描述：下级平台作为服务端向上级平台客户端返回从链路连接应答消息，上级平台在接收到该应答消息结果后，根据结果进行链路连接处理
type DownConnectRsp struct {
	*headerSetter
	Result byte // 0x00:成功； 0x01: VERIFY_CODE错误；0x02:资源紧张，稍后再连接（已经占用）；0x03:其他
}

func NewDownConnectRsp() *DownConnectRsp {
	p := &DownConnectRsp{}
	p.headerSetter = newHeaderSeter(DOWN_CONNECT_RSP)
	return p
}

func (p DownConnectRsp) LinkType() LinkType {
	return DownLinkOnly
}

func (p DownConnectRsp) String() string {
	return fmt.Sprintf("DownConnectRsp{Header:%s Result:%d}", p.Header(), p.Result)
}
