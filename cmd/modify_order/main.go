package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdmodifyorder"
	"google.golang.org/protobuf/proto"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <order_id> <new_price> <new_qty>")
		os.Exit(1)
	}

	orderID, _ := strconv.ParseUint(os.Args[1], 10, 64)
	newPrice, _ := strconv.ParseFloat(os.Args[2], 64)
	newQty, _ := strconv.ParseFloat(os.Args[3], 64)

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
		if acc.GetTrdEnv() == int32(trdcommon.TrdEnv_TrdEnv_Simulate) {
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

	// 2. Modify Order
	trdEnv := int32(trdcommon.TrdEnv_TrdEnv_Simulate)
	trdMarket := int32(trdcommon.TrdMarket_TrdMarket_US)
	op := int32(trdcommon.ModifyOrderOp_ModifyOrderOp_Normal)

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

	modifyReq := trdmodifyorder.Request{
		C2S: &trdmodifyorder.C2S{
			PacketID:      packetID,
			Header:        header,
			OrderID:       &orderID,
			ModifyOrderOp: &op,
			Price:         &newPrice,
			Qty:           &newQty,
		},
	}

	fmt.Printf("Modifying order %d: New Price %.2f, New Qty %.2f\n", orderID, newPrice, newQty)
	conn.SendProto(gofutuapi.TRD_MODIFYORDER, &modifyReq)
	reply, err = conn.NextReplyPacket()
	if err != nil {
		log.Fatalf("Modify order failed: %v", err)
	}

	var modifyResp trdmodifyorder.Response
	if err := proto.Unmarshal(reply.Payload, &modifyResp); err != nil {
		log.Fatalf("Unmarshal modify response failed: %v", err)
	}

	if modifyResp.GetRetType() != 0 {
		log.Fatalf("Modify failed: %s", modifyResp.GetRetMsg())
	}

	fmt.Printf("Order %d modified successfully!\n", modifyResp.GetS2C().GetOrderID())
}
