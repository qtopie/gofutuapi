package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common"
	"github.com/qtopie/gofutuapi/gen/common/getuserinfo"
	"github.com/qtopie/gofutuapi/gen/common/notify"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/qotrequesthistorykl"
	"github.com/qtopie/gofutuapi/gen/qot/qotsub"
	"github.com/qtopie/gofutuapi/gen/qot/qotupdatebasicqot"
	"github.com/qtopie/gofutuapi/gen/qot/qotupdatekl"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetfunds"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetorderlist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetpositionlist"
	"google.golang.org/protobuf/proto"
)

type KLinePoint struct {
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type KLineSeries struct {
	Label  string       `json:"label"`
	KlType string       `json:"klType"`
	Points []KLinePoint `json:"points"`
}

type KLineExport struct {
	Symbol      string        `json:"symbol"`
	Market      string        `json:"market"`
	GeneratedAt string        `json:"generatedAt"`
	Series      []KLineSeries `json:"series"`
	Trade       *TradeSummary `json:"tradeSummary,omitempty"`
}

type PendingOrder struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Qty        float64 `json:"qty"`
	Price      float64 `json:"price"`
	Status     string  `json:"status"`
	StatusCode int32   `json:"statusCode"`
}

type PositionItem struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Price float64 `json:"price"`
	Cost  float64 `json:"cost"`
	Val   float64 `json:"val"`
}

type FundsSummary struct {
	Currency     string  `json:"currency"`
	Balance      float64 `json:"balance"`
	Trading      float64 `json:"trading"`
	Total        float64 `json:"total"`
	TradingRatio float64 `json:"tradingRatio"`
}

type AccountSummary struct {
	AccID         uint64         `json:"accId"`
	TrdEnv        string         `json:"trdEnv"`
	PendingOrders []PendingOrder `json:"pendingOrders"`
	Positions     []PositionItem `json:"positions"`
	Funds         FundsSummary   `json:"funds"`
}

type TradeSummary struct {
	Accounts []AccountSummary `json:"accounts"`
	Errors   []string         `json:"errors"`
}

func marketLabel(market int32) string {
	switch market {
	case int32(qotcommon.QotMarket_QotMarket_HK_Security):
		return "HK"
	case int32(qotcommon.QotMarket_QotMarket_US_Security):
		return "US"
	case int32(qotcommon.QotMarket_QotMarket_CNSH_Security):
		return "SH"
	case int32(qotcommon.QotMarket_QotMarket_CNSZ_Security):
		return "SZ"
	default:
		return fmt.Sprintf("%d", market)
	}
}

func mapKLinePoints(klList []*qotcommon.KLine) []KLinePoint {
	points := make([]KLinePoint, 0, len(klList))
	for _, kl := range klList {
		if kl.GetIsBlank() {
			continue
		}
		points = append(points, KLinePoint{
			Time:   kl.GetTime(),
			Open:   kl.GetOpenPrice(),
			High:   kl.GetHighPrice(),
			Low:    kl.GetLowPrice(),
			Close:  kl.GetClosePrice(),
			Volume: kl.GetVolume(),
		})
	}
	return points
}

func requestHistoryKLines(conn *gofutuapi.FutuApiConn, security *qotcommon.Security, rehabType int32, klType int32, beginTime string, endTime string) ([]*qotcommon.KLine, error) {
	maxAckKLNum := int32(1000)
	var all []*qotcommon.KLine
	var nextReqKey []byte

	for {
		req := qotrequesthistorykl.Request{
			C2S: &qotrequesthistorykl.C2S{
				RehabType:   &rehabType,
				KlType:      &klType,
				Security:    security,
				BeginTime:   &beginTime,
				EndTime:     &endTime,
				MaxAckKLNum: &maxAckKLNum,
				NextReqKey:  nextReqKey,
			},
		}
		conn.SendProto(gofutuapi.QOT_REQUESTHISTORYKL, &req)
		reply, err := conn.NextReplyPacket()
		if err != nil {
			return nil, err
		}
		var resp qotrequesthistorykl.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			return nil, err
		}
		if resp.GetRetType() != 0 {
			return nil, fmt.Errorf("history kl failed: %s", resp.GetRetMsg())
		}
		s2c := resp.GetS2C()
		if s2c == nil {
			break
		}
		all = append(all, s2c.GetKlList()...)
		nextReqKey = s2c.GetNextReqKey()
		if len(nextReqKey) == 0 {
			break
		}
	}

	return all, nil
}

func tradeCurrencyLabel(currency int32) string {
	switch currency {
	case int32(trdcommon.Currency_Currency_HKD):
		return "HKD"
	case int32(trdcommon.Currency_Currency_USD):
		return "USD"
	case int32(trdcommon.Currency_Currency_CNH):
		return "CNH"
	case int32(trdcommon.Currency_Currency_JPY):
		return "JPY"
	case int32(trdcommon.Currency_Currency_SGD):
		return "SGD"
	case int32(trdcommon.Currency_Currency_AUD):
		return "AUD"
	case int32(trdcommon.Currency_Currency_CAD):
		return "CAD"
	case int32(trdcommon.Currency_Currency_MYR):
		return "MYR"
	default:
		return ""
	}
}

func orderStatusLabel(status int32) string {
	if name, ok := trdcommon.OrderStatus_name[status]; ok {
		return name
	}
	return fmt.Sprintf("%d", status)
}

func trdEnvLabel(env int32) string {
	switch env {
	case int32(trdcommon.TrdEnv_TrdEnv_Real):
		return "REAL"
	case int32(trdcommon.TrdEnv_TrdEnv_Simulate):
		return "SIMULATE"
	default:
		return fmt.Sprintf("%d", env)
	}
}

func fetchTradeSummary(conn *gofutuapi.FutuApiConn) *TradeSummary {
	summary := &TradeSummary{}
	userID := uint64(0)
	trdCategory := int32(trdcommon.TrdCategory_TrdCategory_Security)
	needGeneralSecAccount := false

	accReq := trdgetacclist.Request{
		C2S: &trdgetacclist.C2S{
			UserID:                &userID,
			TrdCategory:           &trdCategory,
			NeedGeneralSecAccount: &needGeneralSecAccount,
		},
	}
	conn.SendProto(gofutuapi.TRD_GETACCLIST, &accReq)
	reply, err := conn.NextReplyPacket()
	if err != nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("get acc list failed: %v", err))
		return summary
	}
	var accResp trdgetacclist.Response
	if err := proto.Unmarshal(reply.Payload, &accResp); err != nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("get acc list unmarshal failed: %v", err))
		return summary
	}
	if accResp.GetRetType() != 0 {
		summary.Errors = append(summary.Errors, fmt.Sprintf("get acc list failed: %s", accResp.GetRetMsg()))
		return summary
	}
	accList := accResp.GetS2C().GetAccList()
	if len(accList) == 0 {
		summary.Errors = append(summary.Errors, "no trading account found")
		return summary
	}

	for _, acc := range accList {
		accSum := AccountSummary{
			AccID:  acc.GetAccID(),
			TrdEnv: trdEnvLabel(acc.GetTrdEnv()),
		}

		trdEnv := acc.GetTrdEnv()
		accID := acc.GetAccID()
		trdMarket := int32(trdcommon.TrdMarket_TrdMarket_Unknown)
		if len(acc.GetTrdMarketAuthList()) > 0 {
			trdMarket = acc.GetTrdMarketAuthList()[0]
		}
		header := &trdcommon.TrdHeader{
			TrdEnv:    &trdEnv,
			AccID:     &accID,
			TrdMarket: &trdMarket,
		}

		refreshCache := true
		orderReq := trdgetorderlist.Request{
			C2S: &trdgetorderlist.C2S{
				Header:           header,
				FilterStatusList: gofutuapi.PendingOrderStatuses(),
				RefreshCache:     &refreshCache,
			},
		}
		conn.SendProto(gofutuapi.TRD_GETORDERLIST, &orderReq)
		reply, err = conn.NextReplyPacket()
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get order list failed: %v", accID, err))
		} else {
			var orderResp trdgetorderlist.Response
			if err := proto.Unmarshal(reply.Payload, &orderResp); err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get order list unmarshal failed: %v", accID, err))
			} else if orderResp.GetRetType() != 0 {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get order list failed: %s", accID, orderResp.GetRetMsg()))
			} else {
				for _, order := range orderResp.GetS2C().GetOrderList() {
					accSum.PendingOrders = append(accSum.PendingOrders, PendingOrder{
						Code:       order.GetCode(),
						Name:       order.GetName(),
						Qty:        order.GetQty(),
						Price:      order.GetPrice(),
						Status:     orderStatusLabel(order.GetOrderStatus()),
						StatusCode: order.GetOrderStatus(),
					})
				}
			}
		}

		posReq := trdgetpositionlist.Request{
			C2S: &trdgetpositionlist.C2S{
				Header:       header,
				RefreshCache: &refreshCache,
			},
		}
		conn.SendProto(gofutuapi.TRD_GETPOSITIONLIST, &posReq)
		reply, err = conn.NextReplyPacket()
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get position list failed: %v", accID, err))
		} else {
			var posResp trdgetpositionlist.Response
			if err := proto.Unmarshal(reply.Payload, &posResp); err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get position list unmarshal failed: %v", accID, err))
			} else if posResp.GetRetType() != 0 {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get position list failed: %s", accID, posResp.GetRetMsg()))
			} else {
				for _, position := range posResp.GetS2C().GetPositionList() {
					accSum.Positions = append(accSum.Positions, PositionItem{
						Code:  position.GetCode(),
						Name:  position.GetName(),
						Qty:   position.GetQty(),
						Price: position.GetPrice(),
						Cost:  position.GetAverageCostPrice(),
						Val:   position.GetVal(),
					})
				}
			}
		}

		fundsReq := trdgetfunds.Request{
			C2S: &trdgetfunds.C2S{
				Header:       header,
				RefreshCache: &refreshCache,
			},
		}
		conn.SendProto(gofutuapi.TRD_GETFUNDS, &fundsReq)
		reply, err = conn.NextReplyPacket()
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get funds failed: %v", accID, err))
		} else {
			var fundsResp trdgetfunds.Response
			if err := proto.Unmarshal(reply.Payload, &fundsResp); err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get funds unmarshal failed: %v", accID, err))
			} else if fundsResp.GetRetType() != 0 {
				summary.Errors = append(summary.Errors, fmt.Sprintf("acc %d: get funds failed: %s", accID, fundsResp.GetRetMsg()))
			} else {
				funds := fundsResp.GetS2C().GetFunds()
				if funds != nil {
					balance := funds.GetCash()
					trading := funds.GetFrozenCash()
					total := balance + trading
					ratio := 0.0
					if total > 0 {
						ratio = trading / total
					}
					accSum.Funds = FundsSummary{
						Currency:     tradeCurrencyLabel(funds.GetCurrency()),
						Balance:      balance,
						Trading:      trading,
						Total:        total,
						TradingRatio: ratio,
					}
				}
			}
		}

		summary.Accounts = append(summary.Accounts, accSum)
	}

	return summary
}

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

	// 查询阿里巴巴港股历史K线并导出JSON
	hkMarket := int32(qotcommon.QotMarket_QotMarket_HK_Security)
	alibabaCode := "09988"
	alibabaSecurity := qotcommon.Security{
		Market: &hkMarket,
		Code:   &alibabaCode,
	}

	tradeSummary := fetchTradeSummary(conn)
	subOrUnSub := true
	regOrUnRegPush := true
	firstPush := false
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
				int32(qotcommon.SubType_SubType_KL_Month),
				int32(qotcommon.SubType_SubType_KL_Qurater),
			},
			IsSubOrUnSub:         &subOrUnSub,
			IsRegOrUnRegPush:     &regOrUnRegPush,
			RegPushRehabTypeList: []int32{int32(qotcommon.RehabType_RehabType_Forward)},
			IsFirstPush:          &firstPush,
			IsUnsubAll:           &unSubAll,
			IsSubOrderBookDetail: &subOrderBookDetail,
			ExtendedTime:         &extendedTime,
			Session:              &session,
		},
	}
	conn.SendProto(gofutuapi.QOT_SUB, &qotSubReq)
	reply, err = conn.NextReplyPacket()
	if err != nil {
		log.Println("failed to get qot sub reply", err)
	} else {
		var resp qotsub.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
		if resp.GetRetType() != 0 {
			log.Println("qot sub failed", resp.GetRetMsg())
		}
	}

	rehabType := int32(qotcommon.RehabType_RehabType_Forward)
	currentTime := time.Now()
	endTime := currentTime.Format(time.DateOnly)
	begin7Days := currentTime.AddDate(0, 0, -7).Format(time.DateOnly)
	begin30Days := currentTime.AddDate(0, 0, -30).Format(time.DateOnly)
	begin180Days := currentTime.AddDate(0, 0, -180).Format(time.DateOnly)
	begin3Years := currentTime.AddDate(-3, 0, 0).Format(time.DateOnly)

	daily7, err := requestHistoryKLines(conn, &alibabaSecurity, rehabType, int32(qotcommon.KLType_KLType_Day), begin7Days, endTime)
	if err != nil {
		log.Println("failed to get 7d daily kl", err)
	}
	daily30, err := requestHistoryKLines(conn, &alibabaSecurity, rehabType, int32(qotcommon.KLType_KLType_Day), begin30Days, endTime)
	if err != nil {
		log.Println("failed to get 30d daily kl", err)
	}
	monthly180, err := requestHistoryKLines(conn, &alibabaSecurity, rehabType, int32(qotcommon.KLType_KLType_Month), begin180Days, endTime)
	if err != nil {
		log.Println("failed to get 180d monthly kl", err)
	}
	quarterly3y, err := requestHistoryKLines(conn, &alibabaSecurity, rehabType, int32(qotcommon.KLType_KLType_Quarter), begin3Years, endTime)
	if err != nil {
		log.Println("failed to get 3y quarterly kl", err)
	}

	export := KLineExport{
		Symbol:      fmt.Sprintf("%s.%s", marketLabel(hkMarket), alibabaCode),
		Market:      marketLabel(hkMarket),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Trade:       tradeSummary,
		Series: []KLineSeries{
			{
				Label:  "最近7天(日K)",
				KlType: "Day",
				Points: mapKLinePoints(daily7),
			},
			{
				Label:  "最近30天(日K)",
				KlType: "Day",
				Points: mapKLinePoints(daily30),
			},
			{
				Label:  "最近180天(月K)",
				KlType: "Month",
				Points: mapKLinePoints(monthly180),
			},
			{
				Label:  "最近3年(季K)",
				KlType: "Quarter",
				Points: mapKLinePoints(quarterly3y),
			},
		},
	}

	err = os.MkdirAll("_data", 0o755)
	if err != nil {
		log.Println("failed to create _data directory", err)
	}
	payload, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		log.Println("failed to marshal json", err)
	} else {
		outputPath := "_data/kl-data.json"
		err = os.WriteFile(outputPath, payload, 0o644)
		if err != nil {
			log.Println("failed to write json", err)
		} else {
			log.Println("wrote", outputPath)
		}
	}

	fileServer := http.FileServer(http.Dir("_data"))
	server := &http.Server{Addr: ":8000", Handler: fileServer}
	go func() {
		log.Println("serving _data at http://localhost:8000/kl-viewer.html")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("http server error", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("http server shutdown error", err)
	}
	fmt.Println("Main goroutine exiting.")
}
