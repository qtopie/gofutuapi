package gofutuapi

import (
	"context"
	"errors"
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

	// server push packet on receive hook
	pushHook func(protoId ProtoId, response *ProtoResponse)
	// server reply packet queue
	replyQueue chan *ProtoResponse

	connId       uint64
	nextPacketSN int32

	mu    sync.Mutex
	rw    io.ReadWriteCloser
	state int
}

func Open(context context.Context, option FutuApiOption) (*FutuApiConn, error) {
	c := &FutuApiConn{
		Context:    context,
		replyQueue: make(chan *ProtoResponse, 32),
	}

	conn, err := net.DialTimeout("tcp", option.Address, 5*time.Second)
	if err != nil {
		return nil, err
	}

	c.Conn = conn
	c.rw = c.Conn
	c.nextPacketSN = 1

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

func (conn *FutuApiConn) keepalive(interval int) {
	// update ticker time with server response
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

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
func (conn *FutuApiConn) SendProto(protoId ProtoId, req proto.Message) int {
	header := NewHeader()
	header.ProtoID = protoId
	header.ProtoFmtType = 0
	header.ProtoVer = 0
	header.SerialNo = conn.nextPacketSN

	payload, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	header.UpdateBodyInfo(payload)

	data := append(header.ToBytes(), payload...)
	_, err = conn.rw.Write(data)
	if err != nil {
		panic(err)
	}

	switch protoId {
	case INIT_CONNECT:
		log.Println("sent init connection packet to server")
	case KEEP_ALIVE:
		log.Println("sent heartbeat packet to server")
	default:
		log.Println("sent data to server with protoId", protoId)
	}

	// TODO thread-safe
	conn.nextPacketSN++
	return int(conn.nextPacketSN - 1)
}

func (conn *FutuApiConn) RegisterHook(f func(protoId ProtoId, response *ProtoResponse)) {
	conn.pushHook = f
}

func (conn *FutuApiConn) Close() error {
	log.Println("closing connection", conn.connId)
	return conn.Conn.Close()
}

func (conn *FutuApiConn) NextReplyPacket() (*ProtoResponse, error) {
	for afterCh := time.After(5 * time.Second); ; {
		select {
		case d := <-conn.replyQueue:
			return d, nil
		case <-afterCh:
			return nil, errors.New("timeout to read response")
		}
	}
}

func (conn *FutuApiConn) handleResponsePacket() {
	// update ticker time with server response
	ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

	for {
		select {
		case <-conn.Done():
			log.Println("conn is closed")
			return
		case <-ticker.C:
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

			// check response success or not
			retType := bytesToInt32(payload[:4])
			log.Println("responded", retType)

			switch h.ProtoID {
			case INIT_CONNECT:
				// if fail, log and exit
				var resp initconnect.Response
				err = proto.Unmarshal(payload, &resp)
				if err != nil {
					panic(err)
				}
				if *resp.RetType == 0 {
					conn.connId = *resp.S2C.ConnID
					log.Println("inited connection", conn.connId)
					interval := int(*resp.S2C.KeepAliveInterval)
					go conn.keepalive(interval)
				} else {
					log.Fatalln(resp.String())
				}
			case KEEP_ALIVE:
				// if fail, log and try again
				log.Println("responded keep alive packet")
			default:
				if IsPushProto(h.ProtoID) && conn.pushHook != nil {
					conn.pushHook(h.ProtoID, &ProtoResponse{
						Header:  *h,
						Payload: payload,
						RetType: int(retType),
					})
				} else {
					conn.replyQueue <- &ProtoResponse{
						Header:  *h,
						Payload: payload,
						RetType: int(retType),
					}
				}
			}
		}

	}
}
