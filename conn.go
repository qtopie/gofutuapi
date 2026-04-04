package gofutuapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
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

	option FutuApiOption

	// network connections
	net.Conn

	// server push packet on receive hook
	pushHook func(protoId ProtoId, response *ProtoResponse)
	// server reply packet queue
	replyQueue chan *ProtoResponse

	connId       uint64
	nextPacketSN int32

	mu           sync.Mutex
	rw           io.ReadWriteCloser
	isReconnecting uint32
}

func Open(context context.Context, option FutuApiOption) (*FutuApiConn, error) {
	// We use context.Context but naming it appropriately for the SDK
	return openWithRetry(context, option)
}

// openWithRetry is the internal constructor
func openWithRetry(ctx context.Context, option FutuApiOption) (*FutuApiConn, error) {
	c := &FutuApiConn{
		Context:    ctx,
		option:     option,
		replyQueue: make(chan *ProtoResponse, 128),
	}

	if err := c.connect(); err != nil {
		return nil, err
	}

	// read on server response
	go c.handleResponsePacket()

	return c, nil
}

func (conn *FutuApiConn) connect() error {
	nc, err := net.DialTimeout("tcp", conn.option.Address, 5*time.Second)
	if err != nil {
		return err
	}

	conn.Conn = nc
	conn.rw = conn.Conn
	atomic.StoreInt32(&conn.nextPacketSN, 1)

	return conn.initConnect()
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

	// Send without lock because connect() is called during init or under lock
	packetSN := atomic.AddInt32(&conn.nextPacketSN, 1) - 1
	header := NewHeader()
	header.ProtoID = INIT_CONNECT
	header.SerialNo = packetSN
	payload, _ := proto.Marshal(req)
	header.UpdateBodyInfo(payload)
	data := append(header.ToBytes(), payload...)
	_, err := conn.rw.Write(data)
	return err
}

func (conn *FutuApiConn) tryReconnect() {
	if !atomic.CompareAndSwapUint32(&conn.isReconnecting, 0, 1) {
		return
	}
	defer atomic.StoreUint32(&conn.isReconnecting, 0)

	log.Println("connection lost, attempting to reconnect...")
	
	for {
		select {
		case <-conn.Done():
			return
		default:
			if conn.Conn != nil {
				conn.Conn.Close()
			}
			err := conn.connect()
			if err == nil {
				log.Println("reconnected successfully")
				return
			}
			log.Printf("reconnect failed: %v, retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func (conn *FutuApiConn) keepalive(interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-conn.Done():
			return
		case <-ticker.C:
			if atomic.LoadUint32(&conn.isReconnecting) == 1 {
				continue
			}
			unixEpochSeconds := time.Now().Unix()
			keepaliveMsg := &keepalive.C2S{Time: &unixEpochSeconds}
			keepaliveReq := &keepalive.Request{C2S: keepaliveMsg}
			conn.SendProto(KEEP_ALIVE, keepaliveReq)
		}
	}
}

func (conn *FutuApiConn) SendProto(protoId ProtoId, req proto.Message) int {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// If reconnecting, wait a bit
	for i := 0; i < 10 && atomic.LoadUint32(&conn.isReconnecting) == 1; i++ {
		conn.mu.Unlock()
		time.Sleep(500 * time.Millisecond)
		conn.mu.Lock()
	}

	packetSN := atomic.AddInt32(&conn.nextPacketSN, 1) - 1

	header := NewHeader()
	header.ProtoID = protoId
	header.SerialNo = packetSN

	payload, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	header.UpdateBodyInfo(payload)
	data := append(header.ToBytes(), payload...)
	_, err = conn.rw.Write(data)
	if err != nil {
		log.Printf("write error for proto %d: %v", protoId, err)
		go conn.tryReconnect()
	}

	return int(packetSN)
}

func (conn *FutuApiConn) RegisterHook(f func(protoId ProtoId, response *ProtoResponse)) {
	conn.pushHook = f
}

func (conn *FutuApiConn) Close() error {
	if conn.Conn != nil {
		log.Println("closing connection", conn.connId)
		return conn.Conn.Close()
	}
	return nil
}

func (conn *FutuApiConn) NextReplyPacket() (*ProtoResponse, error) {
	afterCh := time.After(10 * time.Second)
	for {
		select {
		case d := <-conn.replyQueue:
			return d, nil
		case <-afterCh:
			return nil, errors.New("timeout to read response")
		case <-conn.Done():
			return nil, errors.New("connection closed")
		}
	}
}

func (conn *FutuApiConn) handleResponsePacket() {
	for {
		select {
		case <-conn.Done():
			return
		default:
			if atomic.LoadUint32(&conn.isReconnecting) == 1 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			buffer := make([]byte, HEADER_SIZE)
			_, err := io.ReadFull(conn.Conn, buffer)
			if err != nil {
				select {
				case <-conn.Done():
					return
				default:
					log.Printf("read header error: %v", err)
					conn.tryReconnect()
					continue
				}
			}

			h := ParseHeader(buffer[:])
			payload := make([]byte, h.BodyLen)
			_, err = io.ReadFull(conn.Conn, payload)
			if err != nil {
				log.Printf("read payload error: %v", err)
				conn.tryReconnect()
				continue
			}

			var m keepalive.Response
			_ = proto.UnmarshalOptions{DiscardUnknown: true, AllowPartial: true}.Unmarshal(payload, &m)
			retType := int32(-400)
			if m.RetType != nil {
				retType = *m.RetType
			}

			switch h.ProtoID {
			case INIT_CONNECT:
				var resp initconnect.Response
				if err := proto.Unmarshal(payload, &resp); err == nil && *resp.RetType == 0 {
					conn.connId = *resp.S2C.ConnID
					log.Println("inited connection with ID", conn.connId)
					interval := int(*resp.S2C.KeepAliveInterval)
					go conn.keepalive(interval)
				}
			case KEEP_ALIVE:
				var resp keepalive.Response
				if err := proto.Unmarshal(payload, &resp); err == nil {
					// Heartbeat ack
				}
			default:
				pr := &ProtoResponse{
					Header:  *h,
					Payload: payload,
					RetType: int(retType),
				}
				if IsPushProto(h.ProtoID) && conn.pushHook != nil {
					conn.pushHook(h.ProtoID, pr)
				} else {
					select {
					case conn.replyQueue <- pr:
					default:
						// Queue full, drop or handle
					}
				}
			}
		}
	}
}
