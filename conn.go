package gofutuapi

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/qtopie/gofutuapi/gen/common/initconnect"
	"github.com/qtopie/gofutuapi/gen/common/keepalive"
	"google.golang.org/protobuf/proto"
)

var (
	clientID            = "gofutuapi"
	clientVer           = int32(0)
	recvNotify          = true
	packetEncAlgo       = int32(-1)
	pushProtoFmt        = int32(0)
	programmingLanguage = "Go"
)

type FutuApiOption struct {
	Address string
	Timeout time.Duration
}

type FutuApiConn struct {
	// parent context
	context.Context

	// network connections
	net.Conn

	replyQueue chan *ProtoResponse
	pushQueue  chan *ProtoResponse

	nextPacketSN chan int

	mu    sync.Mutex
	rw    io.ReadWriteCloser
	state int
}

func Open(context context.Context, option FutuApiOption) (*FutuApiConn, error) {
	c := &FutuApiConn{
		Context: context,
	}

	conn, err := net.DialTimeout("tcp", option.Address, 5*time.Second)
	if err != nil {
		return nil, err
	}

	c.Conn = conn
	c.rw = c.Conn

	err = c.initConnect()
	if err != nil {
		return nil, err
	}

	// read on server response
	go c.handleResponsePacket()

	return c, nil
}

func (conn *FutuApiConn) initConnect() error {
	req := &initconnect.Request{}
	msg := &initconnect.C2S{}
	msg.ClientVer = &clientVer
	msg.ClientID = &clientID
	msg.RecvNotify = &recvNotify
	msg.PacketEncAlgo = &packetEncAlgo
	msg.PushProtoFmt = &pushProtoFmt
	msg.ProgrammingLanguage = &programmingLanguage
	req.C2S = msg

	conn.SendProto(INIT_CONNECT, req)
	return nil
}

func (conn *FutuApiConn) keepalive() {
	// todo update ticker time with server response
	ticker := time.NewTicker(5000 * time.Millisecond)

	for {
		select {
		case <-conn.Done():
			return
		case <-ticker.C:
			unixEpochSeconds := time.Now().Unix()
			keepaliveMsg := &keepalive.C2S{}
			keepaliveMsg.Time = &unixEpochSeconds
			keepaliveReq := &keepalive.Request{}
			keepaliveReq.C2S = keepaliveMsg

			conn.SendProto(KEEP_ALIVE, keepaliveReq)
		}
	}

}

// SendProto sends protobuf data to futu OpenD server
func (conn *FutuApiConn) SendProto(protoId int, req proto.Message) int {
	header := NewHeader()
	header.ProtoID = int32(protoId)
	header.ProtoFmtType = 0
	header.ProtoVer = 0
	header.SerialNo = 1

	payload, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	header.UpdateBodyInfo(payload)

	data := append(header.ToBytes(), payload...)
	n, err := conn.rw.Write(data)
	if err != nil {
		panic(err)
	}
	log.Println("written data to server")

	return n
}

func (conn *FutuApiConn) Close() error {
	return conn.Conn.Close()
}

func (conn *FutuApiConn) NextReplyPacket() *ProtoResponse {
	return <-conn.replyQueue
}

func (conn *FutuApiConn) handleResponsePacket() {
	for {
		select {
		case <-conn.Done():
			return
		default:
			buffer := make([]byte, HEADER_SIZE)
			_, err := io.ReadFull(conn.Conn, buffer)
			if err != nil {
				panic(err)
			}

			h := ParseHeader(buffer[:])
			payload := make([]byte, h.BodyLen)
			_, err = io.ReadFull(conn.Conn, payload)
			if err != nil {
				panic(err)
			}

			if h.ProtoID == INIT_CONNECT {
				// if fail, log and exit
				go conn.keepalive()
				return
			}

			if h.ProtoID == KEEP_ALIVE {
				// if fail, log and try again

				return
			}

			if IsPushProto(int(h.ProtoID)) {
				// TODO: push listener
				// conn.pushQueue <- &ProtoResponse{
				// 	Header:  *h,
				// 	Payload: payload,
				// }
			} else {
				conn.replyQueue <- &ProtoResponse{
					Header:  *h,
					Payload: payload,
				}
			}
		}

	}
}
