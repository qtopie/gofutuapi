package gofutuapi

import (
	"crypto/sha1"
	"encoding/binary"
)

const (
	HEADER_SIZE  = 2 + 4 + 1 + 1 + 4 + 4 + 20 + 8
	szHeaderFlag = "FT"
)

type ProtoHeader struct {
	szHeaderFlag [2]byte
	ProtoID      int32
	ProtoFmtType byte
	ProtoVer     byte
	SerialNo     int32
	bodyLen      int32
	arrBodySHA1  [20]byte
	arrReserved  [8]byte
}

func NewHeader() *ProtoHeader {
	header := &ProtoHeader{}

	header.arrReserved = [8]byte{}
	return header
}

func ParseHeader(data []byte) *ProtoHeader {
	if len(data) != HEADER_SIZE {
		panic("unmatched header size")
	}
	
	header := NewHeader()

	header.ProtoID = bytesToInt32(data[2:6])
	header.ProtoFmtType = data[6]
	header.ProtoVer = data[7]
	header.SerialNo = bytesToInt32(data[8:12])
	header.bodyLen = bytesToInt32(data[12:16])
	copy(header.arrBodySHA1[:], data[16:36])
	copy(header.arrReserved[:], data[36:])
	return header
}

func (h *ProtoHeader) UpdateBodyInfo(b []byte) {
	h.bodyLen = int32(len(b))
	h.arrBodySHA1 = sha1.Sum(b)
}

func (h *ProtoHeader) ToBytes() []byte {
	data := make([]byte, HEADER_SIZE)
	copy(data, szHeaderFlag)
	copy(data[2:6], int32ToBytes(h.ProtoID))
	data[6] = h.ProtoFmtType
	data[7] = h.ProtoVer
	copy(data[8:12], int32ToBytes(h.SerialNo))
	copy(data[12:16], int32ToBytes(h.bodyLen))
	copy(data[16:36], h.arrBodySHA1[:])
	copy(data[36:], h.arrReserved[:])
	return data
}

func int32ToBytes(n int32) []byte {
	b := make([]byte, 4)

	binary.LittleEndian.PutUint32(b, uint32(n))

	return b
}

func bytesToInt32(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b[:]))
}
