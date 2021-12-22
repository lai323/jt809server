package jt809

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"
)

// 00000048 5B 000000 5A 703FE36E 5D 8010000010 5E 7D9CC4900C
// 00000048 5A01 000000 5A02 703FE36E 5E01 8010000010 5E02 7D9CC4900C
func TestEscape(t *testing.T) {
	escaped := mustHexDecodeString("000000485A010000005A02703FE36E5E0180100000105E027D9CC4900C")
	unescaped := mustHexDecodeString("000000485B0000005A703FE36E5D80100000105E7D9CC4900C")

	unescapeRet := UnEscape(escaped)
	if !reflect.DeepEqual(unescapeRet, unescaped) {
		t.Error("UnEscape error", hex.EncodeToString(unescapeRet))
	}

	escapeRet := Escape(unescaped)
	if !reflect.DeepEqual(escapeRet, escaped) {
		t.Error("Escape error", hex.EncodeToString(escapeRet))
	}
}

func TestEncrypt(t *testing.T) {
	toencrypt := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	encrypted := []byte{0xD3, 0x4C, 0x70, 0x78, 0xA7, 0x3A, 0x41}
	var (
		M1  uint32 = 30000000
		IA1 uint32 = 20000000
		IC1 uint32 = 20000000
		key uint32 = 256178
	)

	encryptRet := make([]byte, len(toencrypt))
	copy(encryptRet, toencrypt)
	Encrypt(M1, IA1, IC1, key, encryptRet)
	if !reflect.DeepEqual(encryptRet, encrypted) {
		t.Error("Encrypt error", hex.EncodeToString(encryptRet))
	}

	decryptRet := make([]byte, len(encrypted))
	copy(decryptRet, encrypted)
	Encrypt(M1, IA1, IC1, key, decryptRet)
	if !reflect.DeepEqual(decryptRet, toencrypt) {
		t.Error("Encrypt error", hex.EncodeToString(decryptRet))
	}
}

func TestReadPacket(t *testing.T) {
	pkt := mustHexDecodeString("5B000000480000008510010133EFB80100000100035D")
	toread := bytes.NewBuffer(mustHexDecodeString("5B000000480000008510010133EFB80100000100035DE8B2D37D9CC4900C77DC78F8676527D8AE12243CFB64CC2FBA619AEFAD33ACCB3256F67BFF19DF33097841098665703FE36E"))

	ret, err := ReadPacket(toread)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(ret, pkt) {
		t.Error("ReadPacket error", hex.EncodeToString(ret))
	}
}
