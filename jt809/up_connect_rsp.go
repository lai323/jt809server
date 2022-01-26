package jt809

import "fmt"

// 主链路登录应答消息
// 链路类型：主链路。
// 消息方向：上级平台往下级平台。
// 业务数据类型标识： UP_CONNECT_RSP.
// 描述：上级平台对下级平台登录请求信息进行安全验证后，返回相应的验证结果。
type UpConnectRsp struct {
	*headerSetter
	Result     byte   // 验证结果， 0x00:成功； 0xO1:IP地址不正确； 0xO2:接入码不正确 0x03:用户没有注册； 0xO4:密码错误； 0xO5:资源紧张，稍后再连接（已经占用）； 0x06:其他
	VerifyCode uint32 // 校验码
}

func NewUpConnectRsp() *UpConnectRsp {
	p := &UpConnectRsp{}
	p.headerSetter = newHeaderSeter(UP_CONNECT_RSP)
	return p
}

func (p UpConnectRsp) LinkType() LinkType {
	return UpLinkOnly
}

func (p UpConnectRsp) String() string {
	return fmt.Sprintf("UpConnectRsp{Header:%s Result:%d, VerifyCode:%d}", p.Header(), p.Result, p.VerifyCode)
}
