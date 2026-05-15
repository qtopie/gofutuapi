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

	market := int32(common.QotMarket_QotMarket_HK_Security)
	code := "00700"
	sec := &common.Security{
		Market: &market,
		Code:   &code,
	}

	// 1. Need to subscribe first
	fmt.Println("Subscribing to Ticker, OrderBook, RTData...")
	err = client.Subscribe([]*common.Security{sec}, []int32{
		int32(common.SubType_SubType_Ticker),
		int32(common.SubType_SubType_OrderBook),
		int32(common.SubType_SubType_RT),
	}, true, true)
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	// Wait for subscription to take effect
	time.Sleep(2 * time.Second)

	// 2. Test GetTicker
	tickers, err := client.GetTicker(sec, 5)
	if err != nil {
		fmt.Printf("GetTicker failed: %v\n", err)
	} else {
		fmt.Printf("Fetched %d tickers\n", len(tickers))
		for _, t := range tickers {
			fmt.Printf("  - Time: %s, Price: %.2f, Vol: %d\n", t.GetTime(), t.GetPrice(), t.GetVolume())
		}
	}

	// 3. Test GetOrderBook
	asks, bids, err := client.GetOrderBook(sec, 5)
	if err != nil {
		fmt.Printf("GetOrderBook failed: %v\n", err)
	} else {
		fmt.Printf("Fetched OrderBook: %d asks, %d bids\n", len(asks), len(bids))
		if len(asks) > 0 {
			fmt.Printf("  Best Ask: %.2f (%d)\n", asks[0].GetPrice(), asks[0].GetVolume())
		}
		if len(bids) > 0 {
			fmt.Printf("  Best Bid: %.2f (%d)\n", bids[0].GetPrice(), bids[0].GetVolume())
		}
	}

	// 4. Test GetRTData
	rt, err := client.GetRTData(sec)
	if err != nil {
		fmt.Printf("GetRTData failed: %v\n", err)
	} else {
		fmt.Printf("Fetched %d RT data points\n", len(rt))
		if len(rt) > 0 {
			fmt.Printf("  Latest RT point: Time: %s, Price: %.2f\n", rt[len(rt)-1].GetTime(), rt[len(rt)-1].GetPrice())
		}
	}
}
