package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	userID := uint64(0)
	trdCategory := int32(trdcommon.TrdCategory_TrdCategory_Security)
	accReq := trdgetacclist.Request{
		C2S: &trdgetacclist.C2S{
			UserID:      &userID,
			TrdCategory: &trdCategory,
		},
	}
	conn.SendProto(gofutuapi.TRD_GETACCLIST, &accReq)
	reply, err := conn.NextReplyPacket()
	if err != nil {
		log.Fatalf("Get acc list failed: %v", err)
	}
	var accResp trdgetacclist.Response
	if err := proto.Unmarshal(reply.Payload, &accResp); err != nil {
		log.Fatalf("Unmarshal acc list failed: %v", err)
	}

	fmt.Println("--- Real Trading Accounts Found ---")
	found := false
	for _, acc := range accResp.GetS2C().GetAccList() {
		if acc.GetTrdEnv() == int32(trdcommon.TrdEnv_TrdEnv_Real) {
			found = true
			fmt.Printf("AccID: %d | Markets: %v\n", 
				acc.GetAccID(), 
				acc.GetTrdMarketAuthList())
		}
	}
	if !found {
		fmt.Println("No real trading accounts found.")
	}
}
