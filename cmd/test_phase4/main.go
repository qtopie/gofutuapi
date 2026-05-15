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

	// 1. Test GetCapitalFlow
	fmt.Println("--- Capital Flow ---")
	flow, err := client.GetCapitalFlow(sec)
	if err != nil {
		fmt.Printf("GetCapitalFlow failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d flow items\n", len(flow))
		if len(flow) > 0 {
			fmt.Printf("  Latest Flow: Inflow: %.2f (Time: %s)\n", flow[len(flow)-1].GetInFlow(), flow[len(flow)-1].GetTime())
		}
	}

	// 2. Test GetCapitalDistribution
	fmt.Println("\n--- Capital Distribution ---")
	dist, err := client.GetCapitalDistribution(sec)
	if err != nil {
		fmt.Printf("GetCapitalDistribution failed: %v\n", err)
	} else {
		fmt.Printf("  Capital In (Super): %.2f\n", dist.GetCapitalInSuper())
		fmt.Printf("  Capital Out (Super): %.2f\n", dist.GetCapitalOutSuper())
	}
}

func protoInt32(v int32) *int32 { return &v }
