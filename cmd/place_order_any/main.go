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
	"github.com/qtopie/gofutuapi/gen/trade/trdplaceorder"
	"google.golang.org/protobuf/proto"
)

func main() {
	if len(os.Args) < 7 {
		fmt.Println("Usage: go run main.go <code> <side: BUY|SELL> <qty> <price> <env: REAL|SIMULATE> <acc_id>")
		os.Exit(1)
	}

	code := os.Args[1]
	sideStr := os.Args[2]
	qty, _ := strconv.ParseFloat(os.Args[3], 64)
	price, _ := strconv.ParseFloat(os.Args[4], 64)
	envStr := os.Args[5]
	accID, _ := strconv.ParseUint(os.Args[6], 10, 64)

	var side int32
	if sideStr == "BUY" {
		side = int32(trdcommon.TrdSide_TrdSide_Buy)
	} else {
		side = int32(trdcommon.TrdSide_TrdSide_Sell)
	}

	var trdEnv int32
	if envStr == "REAL" {
		trdEnv = int32(trdcommon.TrdEnv_TrdEnv_Real)
	} else {
		trdEnv = int32(trdcommon.TrdEnv_TrdEnv_Simulate)
	}

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

	trdMarket := int32(trdcommon.TrdMarket_TrdMarket_US)
	orderType := int32(trdcommon.OrderType_OrderType_Normal)

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

	fmt.Printf("Placing REAL order: %s %v %s @ %.2f (Acc: %d)\n", envStr, sideStr, code, price, accID)
	sn := conn.SendProto(gofutuapi.TRD_PLACEORDER, &orderReq)
	reply, err := conn.WaitReply(sn, 10*time.Second)
	if err != nil {
		log.Fatalf("Place order failed: %v", err)
	}

	var orderResp trdplaceorder.Response
	if err := proto.Unmarshal(reply.Payload, &orderResp); err != nil {
		log.Fatalf("Unmarshal order response failed: %v", err)
	}

	if orderResp.GetRetType() != 0 {
		log.Fatalf("Order failed: %s (Did you unlock trade in OpenD GUI?)", orderResp.GetRetMsg())
	}

	fmt.Printf("Order placed successfully! Order ID: %d\n", orderResp.GetS2C().GetOrderID())
}
