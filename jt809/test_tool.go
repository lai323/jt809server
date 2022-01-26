package jt809

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func mustHexDecodeString(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func mustMarshal(p Packet) []byte {
	pktbytes, err := Marshal(p)
	if err != nil {
		panic(err)
	}
	return pktbytes
}

func mustUnmarshal(data []byte) Packet {
	p, err := Unmarshal(data)
	if err != nil {
		panic(err)
	}
	return p
}

func testpacket(t *testing.T, subtest map[Packet][]byte) {
	for p, data := range subtest {
		dataRet := mustMarshal(p)
		if !reflect.DeepEqual(dataRet, data) {
			t.Error("Packet Marshal error", p, strings.ToUpper(hex.EncodeToString(data)), strings.ToUpper(hex.EncodeToString(dataRet)))
		}
		t.Log(p, strings.ToUpper(hex.EncodeToString(dataRet)))

		packetRet := mustUnmarshal(data)
		if !reflect.DeepEqual(packetRet, p) {
			t.Error("Packet Marshal error", p, strings.ToUpper(hex.EncodeToString(data)), packetRet)
		}
		t.Log(strings.ToUpper(hex.EncodeToString(data)), packetRet)
	}
}
