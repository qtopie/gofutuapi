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
	"github.com/qtopie/gofutuapi/gen/common"
	"github.com/qtopie/gofutuapi/gen/common/getuserinfo"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/qotsub"
	"github.com/qtopie/gofutuapi/gen/qot/qotupdatebasicqot"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

	conn.RegisterHook(func(protoId gofutuapi.ProtoId, reply *gofutuapi.ProtoResponse) {
		log.Println("received server push", protoId, reply.Header)
		if reply == nil || reply.RetType != 0 {
			log.Println("received server push with failure packet", reply.SerialNo)
		}
		switch protoId {
		case gofutuapi.QOT_UPDATEBASICQOT:
			var resp qotupdatebasicqot.Response
			err = proto.Unmarshal(reply.Payload, &resp)
			if err != nil {
				panic(err)
			}
			log.Println(resp.String())
		}
	})

	flag := int32(getuserinfo.UserInfoField_UserInfoField_Basic)
	req := getuserinfo.Request{
		C2S: &getuserinfo.C2S{
			Flag: &flag,
		},
	}
	conn.SendProto(1005, &req)
	reply, err := conn.NextReplyPacket()
	if err != nil {
		log.Println(err)
	} else {
		var resp getuserinfo.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		log.Println(resp.String())
	}

	// 查询阿里巴巴港股最新股价
	hkMarket := int32(qotcommon.QotMarket_QotMarket_HK_Security)
	alibabaCode := "09988"
	alibabaSecurity := qotcommon.Security{
		Market: &hkMarket,
		Code:   &alibabaCode,
	}
	subOrUnSub := true
	regOrUnRegPush := true
	firstPush := true
	unSubAll := false
	subOrderBookDetail := false
	extendedTime := false
	session := int32(common.Session_Session_ALL)

	qotSubReq := qotsub.Request{
		C2S: &qotsub.C2S{
			SecurityList: []*qotcommon.Security{
				&alibabaSecurity,
			},
			SubTypeList: []int32{
				int32(qotcommon.SubType_SubType_Basic),
			},
			IsSubOrUnSub:         &subOrUnSub,
			IsRegOrUnRegPush:     &regOrUnRegPush,
			RegPushRehabTypeList: []int32{},
			IsFirstPush:          &firstPush,
			IsUnsubAll:           &unSubAll,
			IsSubOrderBookDetail: &subOrderBookDetail,
			ExtendedTime:         &extendedTime,
			Session:              &session,
		},
	}
	conn.SendProto(gofutuapi.QOT_SUB, &qotSubReq)

	<-ctx.Done()
	fmt.Println("Main goroutine exiting.")
}
