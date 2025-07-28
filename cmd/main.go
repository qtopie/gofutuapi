package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	// Connect to a TCP server (e.g., localhost:8080)
	conn, err := net.DialTimeout("tcp", "localhost:11111", 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	fmt.Println("Connected to TCP server!")

	// You can now send and receive data using conn.Write() and conn.Read()
	// For example, sending a message:
	_, err = conn.Write([]byte("Hello from Go client!"))
	if err != nil {
		log.Printf("Failed to send data: %v", err)
	}
}
