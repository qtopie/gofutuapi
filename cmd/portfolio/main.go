package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetfunds"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetpositionlist"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	client := gofutuapi.NewClient(conn)

	// 1. Get User Info
	userInfo, err := client.GetUserInfo()
	if err != nil {
		log.Printf("GetUserInfo failed: %v", err)
	} else {
		fmt.Printf("Logged in as: %s (ID: %d)\n", userInfo.GetNickName(), userInfo.GetUserID())
	}

	categories := []int32{
		int32(trdcommon.TrdCategory_TrdCategory_Security),
		int32(trdcommon.TrdCategory_TrdCategory_Future),
	}

	foundHoldings := false
	
	for _, category := range categories {
		catStr := "Security"
		if category == int32(trdcommon.TrdCategory_TrdCategory_Future) {
			catStr = "Future"
		}
		
		fmt.Printf("\n--- Scanning Category: %s ---\n", catStr)
		
		userID := uint64(0)
		needGeneral := true // Try setting this to true
		accReq := trdgetacclist.Request{
			C2S: &trdgetacclist.C2S{
				UserID:      &userID,
				TrdCategory: &category,
				NeedGeneralSecAccount: &needGeneral,
			},
		}
		sn := conn.SendProto(gofutuapi.TRD_GETACCLIST, &accReq)
		reply, err := conn.WaitReply(sn, 10*time.Second)
		if err != nil {
			log.Printf("Get acc list for %s failed: %v", catStr, err)
			continue
		}
		var accResp trdgetacclist.Response
		if err := proto.Unmarshal(reply.Payload, &accResp); err != nil {
			log.Printf("Unmarshal acc list for %s failed: %v", catStr, err)
			continue
		}

		accList := accResp.GetS2C().GetAccList()
		fmt.Printf("Found %d accounts\n", len(accList))
		for _, acc := range accList {
			accID := acc.GetAccID()
			env := acc.GetTrdEnv()
			envStr := "Real"
			if env == int32(trdcommon.TrdEnv_TrdEnv_Simulate) {
				envStr = "Simulate"
			}
			
			firm := acc.GetSecurityFirm()
			card := acc.GetCardNum()
			uniCard := acc.GetUniCardNum()
			
			fmt.Printf("\n[Account %d (%s)] Firm: %d, Card: %s, UniCard: %s, Markets: %v\n", accID, envStr, firm, card, uniCard, acc.GetTrdMarketAuthList())

			// Query Funds
			refresh := true
			fundReq := trdgetfunds.Request{
				C2S: &trdgetfunds.C2S{
					Header: &trdcommon.TrdHeader{
						TrdEnv:    &env,
						AccID:     &accID,
						TrdMarket: &acc.TrdMarketAuthList[0],
					},
					RefreshCache: &refresh,
				},
			}
			sn := conn.SendProto(gofutuapi.TRD_GETFUNDS, &fundReq)
			fReply, err := conn.WaitReply(sn, 10*time.Second)
			if err == nil {
				var fResp trdgetfunds.Response
				if proto.Unmarshal(fReply.Payload, &fResp) == nil && fResp.GetRetType() == 0 {
					funds := fResp.GetS2C().GetFunds()
					fmt.Printf("  Funds - Net Assets: %.2f, Market Val: %.2f\n", 
						funds.GetTotalAssets(), funds.GetMarketVal())
				}
			}

			// Query Positions
			for _, tm := range acc.GetTrdMarketAuthList() {
				posReq := trdgetpositionlist.Request{
					C2S: &trdgetpositionlist.C2S{
						Header: &trdcommon.TrdHeader{
							TrdEnv:    &env,
							AccID:     &accID,
							TrdMarket: &tm,
						},
						RefreshCache: &refresh,
					},
				}
				sn = conn.SendProto(gofutuapi.TRD_GETPOSITIONLIST, &posReq)
				pReply, err := conn.WaitReply(sn, 10*time.Second)
				if err == nil {
					var pResp trdgetpositionlist.Response
					if proto.Unmarshal(pReply.Payload, &pResp) == nil && pResp.GetRetType() == 0 {
						positions := pResp.GetS2C().GetPositionList()
						if len(positions) > 0 {
							foundHoldings = true
							fmt.Printf("  [MARKET %d] Found %d positions:\n", tm, len(positions))
							for _, pos := range positions {
								fmt.Printf("    - %s (%s): Qty %.2f, Market Value: %.2f\n",
									pos.GetCode(), pos.GetName(), pos.GetQty(), pos.GetVal())
							}
						}
					}
				}
			}
		}
	}
	
	if !foundHoldings {
		fmt.Println("\n[FINAL] No holdings found.")
	}
}

