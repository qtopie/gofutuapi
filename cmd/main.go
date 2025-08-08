package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
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

func main() {
	conn, err := gofutuapi.Open(nil, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	writer := bufio.NewWriter(conn)

	// construct init msg
	msg := &initconnect.C2S{}
	msg.ClientVer = &clientVer
	msg.ClientID = &clientID
	msg.RecvNotify = &recvNotify
	msg.PacketEncAlgo = &packetEncAlgo
	msg.PushProtoFmt = &pushProtoFmt
	msg.ProgrammingLanguage = &programmingLanguage
	req := &initconnect.Request{}
	req.C2S = msg

	body, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	// body = append([]byte{10, 25}, body...)
	log.Println("body", body)

	header := gofutuapi.NewHeader()
	header.ProtoID = gofutuapi.INIT_CONNECT
	header.ProtoFmtType = 0
	header.ProtoVer = 0
	header.SerialNo = 1
	header.UpdateBodyInfo(body)

	headBytes := header.ToBytes()

	initData := append(headBytes, body...)
	_, err = writer.Write(initData)
	if err != nil {
		panic(err)
	}
	if err = writer.Flush(); err != nil {
		panic(err)
	}
	log.Println("written data to server")

	// Start a goroutine to read from the connection
	// go func() {
	header.ProtoID = 1004
	reader := bufio.NewReader(conn)
	buffer := make([]byte, gofutuapi.HEADER_SIZE) // A buffer for reading from the network
	_, err = io.ReadFull(reader, buffer)
	if err != nil {
		panic(err)
	}

	h := gofutuapi.ParseHeader(buffer[0:gofutuapi.HEADER_SIZE])
	fmt.Println("received", h.ProtoID)

	payload := make([]byte, h.BodyLen)
	_, err = io.ReadFull(reader, payload)
	if err != nil {
		panic(err)
	}
	fmt.Println(payload)

	var resp initconnect.Response
	err = proto.Unmarshal(payload, &resp)
	if err != nil {
		panic(err)
	}
	log.Println(resp.String())

	unixEpochSeconds := time.Now().Unix()
	keepaliveMsg := &keepalive.C2S{}
	keepaliveMsg.Time = &unixEpochSeconds
	keepaliveReq := &keepalive.Request{}
	keepaliveReq.C2S = keepaliveMsg

	body, err = proto.Marshal(keepaliveReq)
	if err != nil {
		panic(err)
	}
	header.SerialNo = 2
	header.UpdateBodyInfo(body)

	headBytes = header.ToBytes()

	initData = append(headBytes, body...)
	_, err = writer.Write(initData)
	if err != nil {
		panic(err)
	}
	if err = writer.Flush(); err != nil {
		panic(err)
	}
	log.Println("written data to server")

	_, err = io.ReadFull(reader, buffer)
	if err != nil {
		panic(err)
	}

	h = gofutuapi.ParseHeader(buffer[0:gofutuapi.HEADER_SIZE])
	fmt.Println("received", h.ProtoID)

	payload = make([]byte, h.BodyLen)
	_, err = io.ReadFull(reader, payload)
	if err != nil {
		panic(err)
	}
	fmt.Println(payload)

	var keepaliveResp keepalive.Response
	err = proto.Unmarshal(payload, &keepaliveResp)
	if err != nil {
		panic(err)
	}
	log.Println(keepaliveResp.String())

	// }()

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
}
