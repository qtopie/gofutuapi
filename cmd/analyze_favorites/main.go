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
	"github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/qotgetusersecuritygroup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to OpenD: %v. Please make sure OpenD is running.", err)
	}
	defer conn.Close()

	client := gofutuapi.NewClient(conn)

	// 1. Get User Security Groups
	groups, err := client.GetUserSecurityGroup(qotgetusersecuritygroup.GroupType_GroupType_All)
	if err != nil {
		log.Fatalf("Failed to get user security groups: %v", err)
	}

	if len(groups) == 0 {
		fmt.Println("No custom security groups found.")
		return
	}

	for _, group := range groups {
		groupName := group.GetGroupName()
		fmt.Printf("\n--- Group: %s ---\n", groupName)

		// 2. Get Securities in Group
		securities, err := client.GetUserSecurity(groupName)
		if err != nil {
			fmt.Printf("Failed to get securities for group %s: %v\n", groupName, err)
			continue
		}

		if len(securities) == 0 {
			fmt.Println("  No stocks in this group.")
			continue
		}

		var securityList []*common.Security
		for _, s := range securities {
			securityList = append(securityList, s.GetBasic().GetSecurity())
		}

		// 3. Get Snapshots
		snapshots, err := client.GetSecuritySnapshot(securityList)
		if err != nil {
			fmt.Printf("  Failed to get snapshots: %v\n", err)
			continue
		}

		fmt.Printf("%-10s %-20s %10s %10s %10s\n", "Code", "Name", "Price", "Change", "Change%")
		for i, snap := range snapshots {
			basic := snap.GetBasic()
			if basic == nil {
				continue
			}
			
			code := fmt.Sprintf("%s.%s", getMarketName(basic.GetSecurity().GetMarket()), basic.GetSecurity().GetCode())
			name := securities[i].GetBasic().GetName()
			curPrice := basic.GetCurPrice()
			lastClose := basic.GetLastClosePrice()
			change := curPrice - lastClose
			changePct := 0.0
			if lastClose != 0 {
				changePct = (change / lastClose) * 100
			}

			fmt.Printf("%-10s %-20s %10.3f %10.3f %9.2f%%\n", 
				code, name, curPrice, change, changePct)
		}
	}
}

func getMarketName(market int32) string {
	switch market {
	case int32(common.QotMarket_QotMarket_HK_Security):
		return "HK"
	case int32(common.QotMarket_QotMarket_US_Security):
		return "US"
	case int32(common.QotMarket_QotMarket_CNSH_Security):
		return "SH"
	case int32(common.QotMarket_QotMarket_CNSZ_Security):
		return "SZ"
	default:
		return fmt.Sprintf("%d", market)
	}
}
