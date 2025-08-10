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
	"github.com/qtopie/gofutuapi/gen/common/notify"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/qotgetkl"
	"github.com/qtopie/gofutuapi/gen/qot/qotrequesthistorykl"
	"github.com/qtopie/gofutuapi/gen/qot/qotsub"
	"github.com/qtopie/gofutuapi/gen/qot/qotupdatebasicqot"
	"github.com/qtopie/gofutuapi/gen/qot/qotupdatekl"
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
		if reply == nil || reply.RetType != 0 {
			log.Println("received server push with failure packet", reply.SerialNo)
			return
		}
		switch protoId {
		case gofutuapi.NOTIFY:
			var resp notify.Response
			err = proto.Unmarshal(reply.Payload, &resp)
			if err != nil {
				panic(err)
			}
			log.Println(resp.String())
		case gofutuapi.QOT_UPDATEBASICQOT:
			var resp qotupdatebasicqot.Response
			err = proto.Unmarshal(reply.Payload, &resp)
			if err != nil {
				panic(err)
			}
			log.Println("qot basic response", resp.String())
		case gofutuapi.QOT_UPDATEKL:
			var resp qotupdatekl.Response
			err = proto.Unmarshal(reply.Payload, &resp)
			if err != nil {
				panic(err)
			}
			log.Println("qot update kl response", resp.String())
		default:
			log.Println("received server push", protoId, reply.Header)
		}
	})

	flag := int32(getuserinfo.UserInfoField_UserInfoField_Basic)
	req := getuserinfo.Request{
		C2S: &getuserinfo.C2S{
			Flag: &flag,
		},
	}
	conn.SendProto(gofutuapi.GET_USER_INFO, &req)
	reply, err := conn.NextReplyPacket()
	if err != nil {
		log.Println(err)
	} else {
		var resp getuserinfo.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		log.Println("get user info response", resp.String())
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
				int32(qotcommon.SubType_SubType_KL_Day),
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

	// K线获取
	rehabType := int32(qotcommon.RehabType_RehabType_Forward)
	klType := int32(qotcommon.KLType_KLType_Day)
	reqNum := int32(1)
	getKlReq := qotgetkl.Request{
		C2S: &qotgetkl.C2S{
			RehabType: &rehabType,
			KlType:    &klType,
			Security:  &alibabaSecurity,
			ReqNum:    &reqNum,
		},
	}
	conn.SendProto(gofutuapi.QOT_GETKL, &getKlReq)
	reply, err = conn.NextReplyPacket()
	if err != nil {
		log.Println("failed to get kl", err)
	} else {
		var resp qotgetkl.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		log.Println("qot get kl response", resp.String())
	}

	// 历史k线获取
	currentTime := time.Now()
	beginTime, endTime := currentTime.Add(-time.Hour*24*30).Format(time.DateOnly), currentTime.Add(-time.Hour*24*7).Format(time.DateOnly)
	log.Println(beginTime, endTime)
	historyKlReq := qotrequesthistorykl.Request{
		C2S: &qotrequesthistorykl.C2S{
			RehabType: &rehabType,
			KlType:    &klType,
			Security:  &alibabaSecurity,
			BeginTime: &beginTime,
			EndTime:   &endTime,
		},
	}
	conn.SendProto(gofutuapi.QOT_REQUESTHISTORYKL, &historyKlReq)
	reply, err = conn.NextReplyPacket()
	if err != nil {
		log.Println("failed to get history kl", err)
	} else {
		var resp qotrequesthistorykl.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		log.Println("qot get history kl response", resp.String())
	}

	<-ctx.Done()
	fmt.Println("Main goroutine exiting.")
}
