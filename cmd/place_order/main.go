package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdplaceorder"
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
	conn.SendProto(gofutuapi.TRD_GETACCLIST, &accReq)
	reply, err := conn.NextReplyPacket()
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

	// 2. Place Limit Order
	trdEnv := int32(trdcommon.TrdEnv_TrdEnv_Simulate)
	trdMarket := int32(trdcommon.TrdMarket_TrdMarket_US)
	side := int32(trdcommon.TrdSide_TrdSide_Buy)
	orderType := int32(trdcommon.OrderType_OrderType_Normal)
	code := "MSFT"
	qty := 10.0
	price := 395.20

	header := &trdcommon.TrdHeader{
		TrdEnv:    &trdEnv,
		AccID:     &accID,
		TrdMarket: &trdMarket,
	}

	connID := conn.GetConnID()
	serialNo := uint32(time.Now().Unix())
	packetID := &common.PacketID{
		ConnID:   &connID,
		SerialNo: &serialNo,
	}

	orderReq := trdplaceorder.Request{
		C2S: &trdplaceorder.C2S{
			PacketID:  packetID,
			Header:    header,
			TrdSide:   &side,
			OrderType: &orderType,
			Code:      &code,
			Qty:       &qty,
			Price:     &price,
			SecMarket: &trdMarket,
		},
	}

	fmt.Printf("Placing order: BUY 10 MSFT @ %.2f (Acc: %d)\n", price, accID)
	conn.SendProto(gofutuapi.TRD_PLACEORDER, &orderReq)
	reply, err = conn.NextReplyPacket()
	if err != nil {
		log.Fatalf("Place order failed: %v", err)
	}

	var orderResp trdplaceorder.Response
	if err := proto.Unmarshal(reply.Payload, &orderResp); err != nil {
		log.Fatalf("Unmarshal order response failed: %v", err)
	}

	if orderResp.GetRetType() != 0 {
		log.Fatalf("Order failed: %s", orderResp.GetRetMsg())
	}

	fmt.Printf("Order placed successfully! Order ID: %s\n", orderResp.GetS2C().GetOrderID())
}
