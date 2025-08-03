package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common/initconnect"
	"google.golang.org/protobuf/proto"
)

const (
	headerSize = 2 + 4 + 1 + 1 + 4 + 4 + 20 + 8
)

func main() {
	// Connect to a TCP server (e.g., localhost:8080)
	conn, err := net.DialTimeout("tcp", "localhost:11111", 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	fmt.Println("Connected to TCP server!")

	reader := bufio.NewReader(conn)
	data, err := reader.Peek(1024) // Peek up to 1024 bytes
	if err != nil {
		// handle error (could be io.EOF or bufio.ErrBufferFull)
	}
	fmt.Printf("Peeked %d bytes\n", len(data))

	writer := bufio.NewWriter(conn)

	// construct init msg
	msg := &initconnect.C2S{}
	body, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	header := gofutuapi.NewHeader()
	header.ProtoID = 1001
	header.ProtoFmtType = 0
	header.ProtoVer = 0
	header.SerialNo = 1
	header.CalcBodyInfo(body)

	headBytes := header.ToBytes()

	initData := append(headBytes, body...)
	_, err = writer.Write(initData)
	if err != nil {
		panic(err)
	}

	// Start a goroutine to read from the connection
	go func() {
		reader := bufio.NewReader(conn)
		for {
			data, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Read error: %v", err)
				return
			}
			fmt.Printf("Received: %s", data)
		}
	}()

	// Main goroutine writes user input to the connection
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type messages to send. Press Ctrl+C to exit.")
	for scanner.Scan() {
		text := scanner.Text() + "\n"
		_, err := conn.Write([]byte(text))
		if err != nil {
			log.Printf("Write error: %v", err)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Input error: %v", err)
	}

}
