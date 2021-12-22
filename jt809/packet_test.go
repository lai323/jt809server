package jt809

import (
	"testing"
)

func TestUpConnectReq(t *testing.T) {
	pktbytes := mustHexDecodeString("5B000000480000008510010133EFB8010000010003E8B2D37D9CC4900C77DC78F8676527D8AE12243CFB64CC2FBA619AEFAD33ACCB3256F67BFF19DF33097841098665703FE36E5D")
	p := NewUpConnectReq()
	h := p.Header()
	h.SerialNo = 133
	h.Encrypt = 1
	h.EncryptKey = 256178
	h.GNSSCenterID = 20180920
	p.UserID = 20180920
	p.Password = FixedLengthString("20180920", 8)
	p.DownLinkIP = FixedLengthString("127.0.0.1", 32)
	p.DownLinkPort = 809

	subtest := map[Packet][]byte{
		p: pktbytes,
	}
	testpacket(t, subtest)

}
