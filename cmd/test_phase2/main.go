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

	// 1. Test GetMarketState
	fmt.Println("--- Market State ---")
	ms, err := client.GetMarketState([]*common.Security{sec})
	if err != nil {
		fmt.Printf("GetMarketState failed: %v\n", err)
	} else {
		for _, info := range ms {
			fmt.Printf("  %s (%s): %v\n", info.GetSecurity().GetCode(), info.GetName(), info.GetMarketState())
		}
	}

	// 2. Test RequestTradeDate
	fmt.Println("\n--- Trade Date ---")
	dates, err := client.RequestTradeDate(market, "2026-05-01", "2026-05-15")
	if err != nil {
		fmt.Printf("RequestTradeDate failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d trade dates\n", len(dates))
		if len(dates) > 0 {
			fmt.Printf("  Latest: %s (Type: %v)\n", dates[len(dates)-1].GetTime(), dates[len(dates)-1].GetTradeDateType())
		}
	}

	// 3. Test GetPlateList
	fmt.Println("\n--- Plate List (Industry) ---")
	plates, err := client.GetPlateList(market, common.PlateSetType_PlateSetType_Industry)
	if err != nil {
		fmt.Printf("GetPlateList failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d industry plates\n", len(plates))
		if len(plates) > 0 {
			fmt.Printf("  Example: %s (%s)\n", plates[0].GetPlate().GetCode(), plates[0].GetName())
			
			// 4. Test GetPlateSecurity
			fmt.Printf("\n--- Plate Security for %s ---\n", plates[0].GetName())
			stocks, err := client.GetPlateSecurity(plates[0].GetPlate(), common.SortField_SortField_Code, true)
			if err != nil {
				fmt.Printf("  GetPlateSecurity failed: %v\n", err)
			} else {
				fmt.Printf("  Found %d stocks in plate\n", len(stocks))
				for i := 0; i < 3 && i < len(stocks); i++ {
					fmt.Printf("    - %s (%s)\n", stocks[i].GetBasic().GetSecurity().GetCode(), stocks[i].GetBasic().GetName())
				}
			}
		}
	}
}

func protoInt32(v int32) *int32 { return &v }
