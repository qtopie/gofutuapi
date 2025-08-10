package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common/getuserinfo"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	flag := int32(getuserinfo.UserInfoField_UserInfoField_Basic)
	req := getuserinfo.Request{
		C2S: &getuserinfo.C2S{
			Flag: &flag,
		},
	}
	conn.SendProto(1005, &req)
	reply, err := conn.NextReplyPacket()
	if err != nil {
		log.Println(err)
	} else {
		var resp getuserinfo.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		log.Println(resp.String())
	}

	<-ctx.Done()
	fmt.Println("Main goroutine exiting.")
}
