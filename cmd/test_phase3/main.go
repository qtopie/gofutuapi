package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
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

	// 1. Get Accounts
	accs, err := client.GetTradeAccounts()
	if err != nil {
		log.Fatalf("GetTradeAccounts failed: %v", err)
	}
	if len(accs) == 0 {
		log.Fatal("No trading accounts found")
	}
	
	acc := accs[0]
	fmt.Printf("Using Account: %d (Market: %v, Env: %v)\n", acc.GetAccID(), acc.TrdMarketAuthList[0], acc.GetTrdEnv())

	// 2. Test GetMaxTrdQtys
	code := "00700"
	fmt.Printf("\n--- Max Trading Quantities for %s ---\n", code)
	maxQtys, err := client.GetMaxTrdQtys(acc, trdcommon.OrderType_OrderType_Normal, code, 450.0, qotcommon.QotMarket_QotMarket_HK_Security)
	if err != nil {
		fmt.Printf("GetMaxTrdQtys failed: %v\n", err)
	} else {
		fmt.Printf("  Max Cash Buy: %.0f\n", maxQtys.GetMaxCashBuy())
		fmt.Printf("  Max Margin Buy: %.0f\n", maxQtys.GetMaxCashAndMarginBuy())
		fmt.Printf("  Max Position Sell: %.0f\n", maxQtys.GetMaxPositionSell())
	}

	// 3. Test GetMarginRatio
	sec := &qotcommon.Security{
		Market: protoInt32(int32(qotcommon.QotMarket_QotMarket_HK_Security)),
		Code:   &code,
	}
	fmt.Printf("\n--- Margin Ratio for %s ---\n", code)
	ratios, err := client.GetMarginRatio(acc, []*qotcommon.Security{sec})
	if err != nil {
		fmt.Printf("GetMarginRatio failed: %v\n", err)
	} else {
		for _, r := range ratios {
			fmt.Printf("  Initial Margin Long: %.2f%%\n", r.GetImLongRatio()*100)
			fmt.Printf("  Initial Margin Short: %.2f%%\n", r.GetImShortRatio()*100)
		}
	}
}

func protoInt32(v int32) *int32 { return &v }
