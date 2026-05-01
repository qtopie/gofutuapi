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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	market := int32(common.QotMarket_QotMarket_US_Security)
	code := "MSFT"
	sec := &common.Security{
		Market: &market,
		Code:   &code,
	}

	snapshots, err := client.GetSecuritySnapshot([]*common.Security{sec})
	if err != nil {
		log.Fatalf("Failed to get snapshot: %v", err)
	}

	if len(snapshots) > 0 {
		basic := snapshots[0].GetBasic()
		fmt.Printf("Current: %.3f, Open: %.3f, High: %.3f, Low: %.3f, LastClose: %.3f\n",
			basic.GetCurPrice(), basic.GetOpenPrice(), basic.GetHighPrice(), basic.GetLowPrice(), basic.GetLastClosePrice())
	} else {
		fmt.Println("No snapshot data found.")
	}
}
