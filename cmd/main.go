package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common/initconnect"
	"google.golang.org/protobuf/proto"
)

const (
	headerSize = 2 + 4 + 1 + 1 + 4 + 4 + 20 + 8
)

var (
	clientID            = ""
	clientVer           = int32(0)
	recvNotify          = true
	packetEncAlgo       = int32(-1)
	pushProtoFmt        = int32(0)
	programmingLanguage = "Go"
)

func main() {
	// Connect to a TCP server (e.g., localhost:8080)
	conn, err := net.DialTimeout("tcp", "localhost:11111", 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	fmt.Println("Connected to TCP server!")

	// Handle signals for graceful shutdown
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

	body, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	// Protobuf Tags
	body = append([]byte{10, 25}, body...)
	log.Println("body", body)

	header := gofutuapi.NewHeader()
	header.ProtoID = 1001
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
	log.Println("written data to channel", initData, len(initData))

	// Start a goroutine to read from the connection
	go func() {
		reader := bufio.NewReader(conn)
		for n := reader.Buffered(); n < gofutuapi.HEADER_SIZE; n = reader.Buffered() {
			if n > 0 {
				fmt.Println("buffered size", n)
			}
			time.Sleep(100 * time.Millisecond)
		}
		data, err := reader.Peek(gofutuapi.HEADER_SIZE)
		if err != nil {
			panic(err)
		}
		h := gofutuapi.ParseHeader(data[0:44])
		fmt.Println(h.ProtoID)
	}()

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
}
