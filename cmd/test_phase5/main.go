package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/qot/common"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := gofutuapi.NewClient(conn)

	market := common.QotMarket_QotMarket_HK_Security
	code := "00700"
	sec := &common.Security{
		Market: protoInt32(int32(market)),
		Code:   &code,
	}

	// 1. Need to subscribe first
	fmt.Println("Subscribing to Broker Queue...")
	err = client.Subscribe([]*common.Security{sec}, []int32{
		int32(common.SubType_SubType_Broker),
	}, true, true)
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	// 2. Test GetBrokerQueue
	fmt.Println("--- Broker Queue ---")
	asks, bids, err := client.GetBrokerQueue(sec)
	if err != nil {
		fmt.Printf("GetBrokerQueue failed: %v\n", err)
	} else {
		fmt.Printf("  Asks: %d, Bids: %d\n", len(asks), len(bids))
		if len(asks) > 0 {
			fmt.Printf("  First Ask Broker: %s (%d)\n", asks[0].GetName(), asks[0].GetId())
		}
	}
}

func protoInt32(v int32) *int32 { return &v }
