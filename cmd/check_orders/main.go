package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetorderlist"
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

	// 1. Get Simulation Account for US market
	userID := uint64(0)
	trdCategory := int32(trdcommon.TrdCategory_TrdCategory_Security)
	accReq := trdgetacclist.Request{
		C2S: &trdgetacclist.C2S{
			UserID:      &userID,
			TrdCategory: &trdCategory,
		},
	}
	sn := conn.SendProto(gofutuapi.TRD_GETACCLIST, &accReq)
	reply, err := conn.WaitReply(sn, 10*time.Second)
	if err != nil {
		log.Fatalf("Get acc list failed: %v", err)
	}
	var accResp trdgetacclist.Response
	if err := proto.Unmarshal(reply.Payload, &accResp); err != nil {
		log.Fatalf("Unmarshal acc list failed: %v", err)
	}

	var accID uint64
	found := false
	for _, acc := range accResp.GetS2C().GetAccList() {
		// Look for Simulation account (TrdEnv_Simulate)
		if acc.GetTrdEnv() == int32(trdcommon.TrdEnv_TrdEnv_Simulate) {
			// Ensure it has US market authority
			for _, market := range acc.GetTrdMarketAuthList() {
				if market == int32(trdcommon.TrdMarket_TrdMarket_US) {
					accID = acc.GetAccID()
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		log.Fatal("No simulation trading account found.")
	}

	// 2. Fetch Orders
	trdEnv := int32(trdcommon.TrdEnv_TrdEnv_Simulate)
	trdMarket := int32(trdcommon.TrdMarket_TrdMarket_US)
	refreshCache := true

	header := &trdcommon.TrdHeader{
		TrdEnv:    &trdEnv,
		AccID:     &accID,
		TrdMarket: &trdMarket,
	}

	orderReq := trdgetorderlist.Request{
		C2S: &trdgetorderlist.C2S{
			Header:       header,
			RefreshCache: &refreshCache,
		},
	}

	sn = conn.SendProto(gofutuapi.TRD_GETORDERLIST, &orderReq)
	reply, err = conn.WaitReply(sn, 10*time.Second)
	if err != nil {
		log.Fatalf("Get orders failed: %v", err)
	}

	var orderResp trdgetorderlist.Response
	if err := proto.Unmarshal(reply.Payload, &orderResp); err != nil {
		log.Fatalf("Unmarshal order list response failed: %v", err)
	}

	if orderResp.GetRetType() != 0 {
		log.Fatalf("Get orders failed: %s", orderResp.GetRetMsg())
	}

	orders := orderResp.GetS2C().GetOrderList()
	fmt.Printf("--- Found %d orders for Account %d ---\n", len(orders), accID)
	
	for _, order := range orders {
		var statusName string
		if name, ok := trdcommon.OrderStatus_name[order.GetOrderStatus()]; ok {
			statusName = name
		} else {
			statusName = fmt.Sprintf("%d", order.GetOrderStatus())
		}
		fmt.Printf("Order ID: %d | %s | %s | Qty: %.2f | Price: %.2f | Status: %s | Remark: %s\n",
			order.GetOrderID(),
			order.GetCode(),
			order.GetOrderType(),
			order.GetQty(),
			order.GetPrice(),
			statusName,
			order.GetRemark(),
		)
	}
}
