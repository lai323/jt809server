package jt809

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/howeyc/crc16"
	"github.com/lai323/bytecodec"
)

const (
	BeginDelimiter  byte = 0x5b
	EndDelimiter    byte = 0x5d
	BeginEscapeChar byte = 0x5a
	EndEscapeChar   byte = 0x5e
)

func Marshal(p Packet) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)
	err := enc.Encode(p)
	return buf.Bytes(), err
}

func Unmarshal(data []byte) (Packet, error) {
	buf := bytes.NewBuffer(data)
	dec := NewDecoder(buf)
	p, err := dec.Decode()
	return p, err
}

type UnsupportPacketErr struct {
	Type uint16
}

func (e *UnsupportPacketErr) Error() string {
	return fmt.Sprintf("jt809 unsupport packet %#04x", e.Type)
}

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: bufio.NewReader(r),
	}
}

func (c *Decoder) Decode() (p Packet, err error) {
	pktbytes, err := ReadPacket(c.r)
	if err != nil {
		return nil, err
	}
	pktbytes, err = RawPacketBytes(pktbytes)

	header := Header{}
	err = bytecodec.Unmarshal(pktbytes[:HeaderLength], &header)
	if err != nil {
		return nil, fmt.Errorf("jt809 Decode Header Unmarshal error: %s", err)
	}
	pktbytes = pktbytes[HeaderLength:]

	if header.Encrypt == 1 {
		var (
			M1  uint32 = 30000000
			IA1 uint32 = 20000000
			IC1 uint32 = 20000000
		)
		Encrypt(M1, IA1, IC1, header.EncryptKey, pktbytes)
	}

	new := newPacketMap[header.Type]
	if new == nil {
		return nil, &UnsupportPacketErr{Type: header.Type}
	}
	pkt := new()
	err = bytecodec.Unmarshal(pktbytes, pkt)
	if err != nil {
		return nil, fmt.Errorf("jt809 Decode Packet Unmarshal error: %s", err)
	}
	pkt.SetHeader(header)
	return pkt, nil
}

type Encoder struct {
	w *bufio.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: bufio.NewWriter(w),
	}
}

func (c *Encoder) Encode(p Packet) error {
	pktbytes, err := bytecodec.Marshal(p)
	if err != nil {
		if err != nil {
			return fmt.Errorf("jt809 Encode Packet Marshal error: %s", err)
		}
	}
	header := p.Header()
	if header.Encrypt == 1 {
		var (
			M1  uint32 = 30000000
			IA1 uint32 = 20000000
			IC1 uint32 = 20000000
		)
		Encrypt(M1, IA1, IC1, header.EncryptKey, pktbytes)
	}

	header.Length = 1 + HeaderLength + uint32(len(pktbytes)) + 1 + 2
	headerbytes, err := bytecodec.Marshal(header)
	if err != nil {
		return fmt.Errorf("jt809 Encode Header Marshal error: %s", err)
	}
	pktbytes = append(headerbytes, pktbytes...)

	sumbyte := make([]byte, 2)
	binary.BigEndian.PutUint16(sumbyte, crc16.ChecksumCCITTFalse(pktbytes))
	pktbytes = append(pktbytes, sumbyte...)

	pktbytes = Escape(pktbytes)
	pktbytes = append([]byte{BeginDelimiter}, pktbytes...)
	pktbytes = append(pktbytes, EndDelimiter)
	_, err = c.w.Write(pktbytes)
	if err != nil {
		return err
	}
	return c.w.Flush()
}

func ReadPacket(r io.Reader) ([]byte, error) {
	var (
		pkt   []byte
		buff  = make([]byte, 1)
		found bool
	)
	for {
		_, err := r.Read(buff)
		if err != nil {
			return nil, err
		}
		if !found && buff[0] == BeginDelimiter {
			found = true
			pkt = append(pkt, buff[0])
			continue
		}
		pkt = append(pkt, buff[0])

		if buff[0] == EndDelimiter {
			break
		}
	}
	return pkt, nil
}

type WrongCRC16CCITTError struct {
	Src    []byte
	Sum    uint16
	Should uint16
}

func (e *WrongCRC16CCITTError) Error() string {
	return fmt.Sprintf("CRC16CCITT check fail src:%#x sum:%#x should be:%#x", e.Src, e.Sum, e.Should)
}

// 返回已验证的，去掉头尾标识符、校验码的 bytes
// 这是一个新的 byte slice 不是原 slice 的切片
// 验证异常返回 &WrongBCCError{dst, bcc, should}
func RawPacketBytes(src []byte) ([]byte, error) {
	src = UnEscape(src[1 : len(src)-1])
	sum := binary.BigEndian.Uint16(src[len(src)-2:])
	content := src[:len(src)-2]
	if should := crc16.ChecksumCCITT(content); sum != should {
		return content, &WrongCRC16CCITTError{Src: src, Sum: sum, Should: should}
	}
	return content, nil
}

// 转义
// 0x5b -> 0x5a, 0x01
// 0x5a -> 0x5a, 0x02
// 0x5d -> 0x5e, 0x01
// 0x5e -> 0x5e, 0x02
func Escape(b []byte) (ret []byte) {
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case BeginDelimiter:
			ret = append(ret, 0x5a, 0x01)
		case BeginEscapeChar:
			ret = append(ret, 0x5a, 0x02)
		case EndDelimiter:
			ret = append(ret, 0x5e, 0x01)
		case EndEscapeChar:
			ret = append(ret, 0x5e, 0x02)
		default:
			ret = append(ret, b[i])
		}
	}
	return ret
}

// 反转义
// 0x5a, 0x01 -> 0x5b
// 0x5a, 0x02 -> 0x5a
// 0x5e, 0x01 -> 0x5d
// 0x5e, 0x02 -> 0x5e
func UnEscape(d []byte) (ret []byte) {
	skip := false
	dlength := len(d)
	for i := 0; i < dlength; i++ {
		if skip {
			skip = false
			continue
		}
		if i+1 == dlength {
			ret = append(ret, d[i])
			break
		}

		switch {
		case d[i] == 0x5a && d[i+1] == 0x01:
			ret = append(ret, BeginDelimiter)
			skip = true
		case d[i] == 0x5a && d[i+1] == 0x02:
			ret = append(ret, BeginEscapeChar)
			skip = true
		case d[i] == 0x5e && d[i+1] == 0x01:
			ret = append(ret, EndDelimiter)
			skip = true
		case d[i] == 0x5e && d[i+1] == 0x02:
			ret = append(ret, EndEscapeChar)
			skip = true
		default:
			ret = append(ret, d[i])
		}
	}
	return ret
}

// Const unsigned uint32_t M1  = A;
// Const unsigned uint32_t IA1 = B;
// Const unsigned uint32_t IC1 = C;
// Void encrypt(uint32_t key, unsigned char*buffer, uint32_t size)
// {
// 	uint32_t idx = 0;
// 	if (key == 0) {
// 		key = 1;
// 	}
// 	while(idx < size){
// 		key = IA1 * (key % M1) + IC1;
// 		buffer[idx++] ^= (unsigned char)((key>>20)&0xff)
// 	}
// }
func Encrypt(M1, IA1, IC1, key uint32, data []byte) {
	if key == 0 {
		key = 1
	}
	for i, v := range data {
		key = IA1*(key%M1) + IC1
		data[i] = v ^ byte((key>>20)&0xff)
	}
}
