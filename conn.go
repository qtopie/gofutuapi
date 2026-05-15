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
	// pending replies map
	pendingReplies map[int32]chan *ProtoResponse
	repliesMu      sync.Mutex

	connId       uint64
	nextPacketSN int32

	mu             sync.Mutex
	rw             io.ReadWriteCloser
	isReconnecting uint32
}

func Open(ctx context.Context, option FutuApiOption) (*FutuApiConn, error) {
	// Using the context from the parameter
	return openWithRetry(ctx, option)
}

func OpenContext(ctx context.Context, option FutuApiOption) (*FutuApiConn, error) {
	return openWithRetry(ctx, option)
}

// openWithRetry is the internal constructor
func openWithRetry(ctx context.Context, option FutuApiOption) (*FutuApiConn, error) {
	c := &FutuApiConn{
		Context:        ctx,
		option:         option,
		pendingReplies: make(map[int32]chan *ProtoResponse),
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

	// In connect, we handle initConnect manually to get connId before starting handleResponsePacket
	return conn.initConnectSync()
}

func (conn *FutuApiConn) initConnectSync() error {
	req := &initconnect.Request{
		C2S: &initconnect.C2S{
			ClientVer:           &clientVer,
			ClientID:            &clientID,
			RecvNotify:          &recvNotify,
			PacketEncAlgo:       &packetEncAlgo,
			PushProtoFmt:        &pushProtoFmt,
			ProgrammingLanguage: &programmingLanguage,
		},
	}

	packetSN := atomic.AddInt32(&conn.nextPacketSN, 1) - 1
	header := NewHeader()
	header.ProtoID = INIT_CONNECT
	header.SerialNo = packetSN
	payload, _ := proto.Marshal(req)
	header.UpdateBodyInfo(payload)
	data := append(header.ToBytes(), payload...)

	if _, err := conn.rw.Write(data); err != nil {
		return err
	}

	// Read reply immediately
	respBuf := make([]byte, HEADER_SIZE)
	if _, err := io.ReadFull(conn.rw, respBuf); err != nil {
		return err
	}
	respHeader := ParseHeader(respBuf)
	respPayload := make([]byte, respHeader.BodyLen)
	if _, err := io.ReadFull(conn.rw, respPayload); err != nil {
		return err
	}

	var resp initconnect.Response
	if err := proto.Unmarshal(respPayload, &resp); err != nil {
		return err
	}

	if resp.GetRetType() != 0 {
		return fmt.Errorf("init connect failed: %s", resp.GetRetMsg())
	}

	conn.connId = resp.GetS2C().GetConnID()
	log.Printf("inited connection with ID %d", conn.connId)

	return nil
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

func (conn *FutuApiConn) SendProto(protoId ProtoId, req proto.Message) int32 {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// If reconnecting, wait a bit
	for i := 0; i < 10 && atomic.LoadUint32(&conn.isReconnecting) == 1; i++ {
		conn.mu.Unlock()
		time.Sleep(500 * time.Millisecond)
		conn.mu.Lock()
	}

	packetSN := atomic.AddInt32(&conn.nextPacketSN, 1) - 1

	// Register pending reply
	replyCh := make(chan *ProtoResponse, 1)
	conn.repliesMu.Lock()
	conn.pendingReplies[packetSN] = replyCh
	conn.repliesMu.Unlock()

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

	return packetSN
}

func (conn *FutuApiConn) RegisterHook(f func(protoId ProtoId, response *ProtoResponse)) {
	conn.pushHook = f
}

func (conn *FutuApiConn) GetConnID() uint64 {
	return conn.connId
}

func (conn *FutuApiConn) Close() error {
	if conn.Conn != nil {
		log.Println("closing connection", conn.connId)
		return conn.Conn.Close()
	}
	return nil
}

func (conn *FutuApiConn) WaitReply(sn int32, timeout time.Duration) (*ProtoResponse, error) {
	conn.repliesMu.Lock()
	ch, ok := conn.pendingReplies[sn]
	conn.repliesMu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no pending reply for SN %d", sn)
	}

	defer func() {
		conn.repliesMu.Lock()
		delete(conn.pendingReplies, sn)
		conn.repliesMu.Unlock()
	}()

	select {
	case d := <-ch:
		return d, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for reply SN %d", sn)
	case <-conn.Done():
		return nil, errors.New("connection closed")
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
			_, err := io.ReadFull(conn.rw, buffer)
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
			_, err = io.ReadFull(conn.rw, payload)
			if err != nil {
				log.Printf("read payload error: %v", err)
				conn.tryReconnect()
				continue
			}

			resp := &ProtoResponse{
				Header:  *h,
				Payload: payload,
			}

			// check if push or reply
			if h.SerialNo == 0 || IsPushProto(h.ProtoID) {
				if conn.pushHook != nil {
					conn.pushHook(h.ProtoID, resp)
				}
			} else {
				conn.repliesMu.Lock()
				ch, ok := conn.pendingReplies[h.SerialNo]
				if ok {
					select {
					case ch <- resp:
					default:
						log.Printf("reply channel full for SN %d", h.SerialNo)
					}
				} else {
					// Some replies might not be waited for, but we should log if it's unexpected
					// log.Printf("unexpected reply packet SN %d ProtoID %d", h.SerialNo, h.ProtoID)
				}
				conn.repliesMu.Unlock()
			}
		}
	}
}
