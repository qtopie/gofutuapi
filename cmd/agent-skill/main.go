package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/qotsetpricereminder"
	"github.com/qtopie/gofutuapi/gen/qot/qotstockfilter"
	"github.com/qtopie/gofutuapi/gen/trade/trdupdateorder"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"google.golang.org/protobuf/proto"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	cmd := flag.String("cmd", "portfolio", "command to run: portfolio, orders, accounts, modify, place-order, snapshot, set-reminder, watch, filter, fills, history-kl, option-expiration, option-chain")
	marketFlag := flag.String("market", "HK", "market: HK, US, SH, SZ")
	envFlag := flag.String("env", "REAL", "environment: REAL, SIMULATE")
	accID := flag.Uint64("acc-id", 0, "specific account ID")
	orderID := flag.String("order-id", "", "order ID to modify")
	price := flag.Float64("price", 0, "new price or order price")
	qty := flag.Float64("qty", 0, "new quantity or order quantity")
	op := flag.Int("op", 1, "modify op: 1 for Normal, 2 for Cancel")
	side := flag.Int("side", 1, "trd side: 1 Buy, 2 Sell")
	orderType := flag.Int("type", 1, "order type: 1 Normal")
	code := flag.String("code", "", "security code(s)")
	remindType := flag.Int("remind-type", 1, "PriceReminderType: 1 PriceUp, 2 PriceDown...")
	remindFreq := flag.Int("remind-freq", 1, "PriceReminderFreq: 1 Always, 2 OnceADay...")
	remindValue := flag.Float64("remind-value", 0, "reminder trigger value")
	remindNote := flag.String("remind-note", "", "note for reminder")
	filterField := flag.Int("filter-field", 0, "StockField for filter")
	filterMin := flag.Float64("filter-min", 0, "min value for filter")
	filterMax := flag.Float64("filter-max", 0, "max value for filter")
	beginTime := flag.String("begin", "", "begin time for history kl or option chain")
	endTime := flag.String("end", "", "end time for history kl or option chain")
	klTypeStr := flag.String("kl-type", "DAY", "KLType: DAY, WEEK, MONTH, 1M, 5M, 15M, 30M, 60M")
	rehabTypeStr := flag.String("rehab-type", "FORWARD", "RehabType: NONE, FORWARD, BACKWARD")
	optTypeStr := flag.String("opt-type", "CALL", "OptionType: CALL, PUT, ALL")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		sendError(fmt.Errorf("failed to connect to OpenD: %w", err))
		return
	}
	defer conn.Close()

	client := gofutuapi.NewClient(conn)

	var trdEnv trdcommon.TrdEnv
	if strings.ToUpper(*envFlag) == "REAL" {
		trdEnv = trdcommon.TrdEnv_TrdEnv_Real
	} else {
		trdEnv = trdcommon.TrdEnv_TrdEnv_Simulate
	}

	var trdMarket trdcommon.TrdMarket
	if strings.ToUpper(*marketFlag) == "US" {
		trdMarket = trdcommon.TrdMarket_TrdMarket_US
	} else {
		trdMarket = trdcommon.TrdMarket_TrdMarket_HK
	}

	switch *cmd {
	case "accounts":
		handleAccounts(client)
	case "portfolio":
		handlePortfolio(client, trdEnv, trdMarket, *accID)
	case "orders":
		handleOrders(client, trdEnv, trdMarket, *accID)
	case "modify":
		handleModify(client, trdEnv, trdMarket, *accID, *orderID, *price, *qty, *op)
	case "place-order":
		handlePlaceOrder(client, trdEnv, trdMarket, *accID, trdcommon.TrdSide(*side), trdcommon.OrderType(*orderType), *code, *qty, *price)
	case "snapshot":
		handleSnapshot(client, *code)
	case "set-reminder":
		handleSetReminder(client, *code, *remindType, *remindFreq, *remindValue, *remindNote)
	case "watch":
		handleWatch(client)
	case "filter":
		handleFilter(client, *marketFlag, *filterField, *filterMin, *filterMax)
	case "fills":
		handleFills(client, trdEnv, trdMarket, *accID)
	case "history-kl":
		handleHistoryKL(client, *code, *beginTime, *endTime, *klTypeStr, *rehabTypeStr)
	case "option-expiration":
		handleOptionExpiration(client, *code)
	case "option-chain":
		handleOptionChain(client, *code, *beginTime, *endTime, *optTypeStr)
	default:
		sendError(fmt.Errorf("unknown command: %s", *cmd))
	}
}

func parseKLType(s string) qotcommon.KLType {
	switch strings.ToUpper(s) {
	case "1M":
		return qotcommon.KLType_KLType_1Min
	case "DAY":
		return qotcommon.KLType_KLType_Day
	case "WEEK":
		return qotcommon.KLType_KLType_Week
	case "MONTH":
		return qotcommon.KLType_KLType_Month
	case "YEAR":
		return qotcommon.KLType_KLType_Year
	case "5M":
		return qotcommon.KLType_KLType_5Min
	case "15M":
		return qotcommon.KLType_KLType_15Min
	case "30M":
		return qotcommon.KLType_KLType_30Min
	case "60M":
		return qotcommon.KLType_KLType_60Min
	case "3M":
		return qotcommon.KLType_KLType_3Min
	case "QUARTER":
		return qotcommon.KLType_KLType_Quarter
	default:
		return qotcommon.KLType_KLType_Day
	}
}

func parseRehabType(s string) qotcommon.RehabType {
	switch strings.ToUpper(s) {
	case "NONE":
		return qotcommon.RehabType_RehabType_None
	case "FORWARD":
		return qotcommon.RehabType_RehabType_Forward
	case "BACKWARD":
		return qotcommon.RehabType_RehabType_Backward
	default:
		return qotcommon.RehabType_RehabType_Forward
	}
}

func parseOptionType(s string) qotcommon.OptionType {
	switch strings.ToUpper(s) {
	case "CALL":
		return qotcommon.OptionType_OptionType_Call
	case "PUT":
		return qotcommon.OptionType_OptionType_Put
	case "ALL":
		return qotcommon.OptionType_OptionType_Unknown
	default:
		return qotcommon.OptionType_OptionType_Unknown
	}
}

func handleAccounts(client *gofutuapi.FutuClient) {
	accounts, err := client.GetTradeAccounts()
	if err != nil {
		sendError(err)
		return
	}
	sendSuccess(accounts)
}

func handlePortfolio(client *gofutuapi.FutuClient, env trdcommon.TrdEnv, market trdcommon.TrdMarket, accID uint64) {
	var acc *trdcommon.TrdAcc
	var err error

	if accID != 0 {
		accounts, err := client.GetTradeAccounts()
		if err != nil {
			sendError(err)
			return
		}
		for _, a := range accounts {
			if a.GetAccID() == accID {
				acc = a
				break
			}
		}
	} else {
		acc, err = client.FindTradeAccount(env, market)
	}

	if err != nil || acc == nil {
		sendError(fmt.Errorf("account not found for env=%v market=%v id=%d", env, market, accID))
		return
	}

	funds, err := client.GetFundsForAccount(acc, true)
	if err != nil {
		sendError(err)
		return
	}

	positions, err := client.GetPositionsForAccount(acc, true)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"account":   acc,
		"funds":     funds,
		"positions": positions,
	})
}

func handleOrders(client *gofutuapi.FutuClient, env trdcommon.TrdEnv, market trdcommon.TrdMarket, accID uint64) {
	var acc *trdcommon.TrdAcc
	var err error

	if accID != 0 {
		accounts, err := client.GetTradeAccounts()
		if err != nil {
			sendError(err)
			return
		}
		for _, a := range accounts {
			if a.GetAccID() == accID {
				acc = a
				break
			}
		}
	} else {
		acc, err = client.FindTradeAccount(env, market)
	}

	if err != nil || acc == nil {
		sendError(fmt.Errorf("account not found"))
		return
	}

	orders, err := client.GetOrderListForAccount(acc, nil, true)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"account": acc,
		"orders":  orders,
	})
}

func handleFills(client *gofutuapi.FutuClient, env trdcommon.TrdEnv, market trdcommon.TrdMarket, accID uint64) {
	var acc *trdcommon.TrdAcc
	var err error

	if accID != 0 {
		accounts, err := client.GetTradeAccounts()
		if err != nil {
			sendError(err)
			return
		}
		for _, a := range accounts {
			if a.GetAccID() == accID {
				acc = a
				break
			}
		}
	} else {
		acc, err = client.FindTradeAccount(env, market)
	}

	if err != nil || acc == nil {
		sendError(fmt.Errorf("account not found"))
		return
	}

	fills, err := client.GetOrderFillListForAccount(acc, true)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"account": acc,
		"fills":   fills,
	})
}

func handleModify(client *gofutuapi.FutuClient, env trdcommon.TrdEnv, market trdcommon.TrdMarket, accID uint64, orderID string, price float64, qty float64, op int) {
	var acc *trdcommon.TrdAcc
	var err error

	if accID != 0 {
		accounts, err := client.GetTradeAccounts()
		if err != nil {
			sendError(err)
			return
		}
		for _, a := range accounts {
			if a.GetAccID() == accID {
				acc = a
				break
			}
		}
	} else {
		acc, err = client.FindTradeAccount(env, market)
	}

	if err != nil || acc == nil {
		sendError(fmt.Errorf("account not found"))
		return
	}

	err = client.ModifyOrder(acc, orderID, price, qty, trdcommon.ModifyOrderOp(op))
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"orderID": orderID,
		"status":  "success",
	})
}

func handlePlaceOrder(client *gofutuapi.FutuClient, env trdcommon.TrdEnv, market trdcommon.TrdMarket, accID uint64, side trdcommon.TrdSide, oType trdcommon.OrderType, code string, qty float64, price float64) {
	var acc *trdcommon.TrdAcc
	var err error

	if accID != 0 {
		accounts, err := client.GetTradeAccounts()
		if err != nil {
			sendError(err)
			return
		}
		for _, a := range accounts {
			if a.GetAccID() == accID {
				acc = a
				break
			}
		}
	} else {
		acc, err = client.FindTradeAccount(env, market)
	}

	if err != nil || acc == nil {
		sendError(fmt.Errorf("account not found"))
		return
	}

	orderIDEx, orderID, err := client.PlaceOrder(acc, side, oType, code, qty, price)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"orderID":   orderID,
		"orderIDEx": orderIDEx,
		"status":    "submitted",
	})
}

func handleSnapshot(client *gofutuapi.FutuClient, codes string) {
	if codes == "" {
		sendError(fmt.Errorf("code is required for snapshot"))
		return
	}

	codeList := strings.Split(codes, ",")
	var securityList []*qotcommon.Security
	for _, c := range codeList {
		market, symbol, found := strings.Cut(strings.TrimSpace(c), ".")
		if !found {
			sendError(fmt.Errorf("invalid code format: %s", c))
			return
		}
		var m int32
		switch strings.ToUpper(market) {
		case "HK":
			m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
		case "US":
			m = int32(qotcommon.QotMarket_QotMarket_US_Security)
		case "SH":
			m = int32(qotcommon.QotMarket_QotMarket_CNSH_Security)
		case "SZ":
			m = int32(qotcommon.QotMarket_QotMarket_CNSZ_Security)
		default:
			sendError(fmt.Errorf("unsupported market: %s", market))
			return
		}
		securityList = append(securityList, &qotcommon.Security{
			Market: &m,
			Code:   &symbol,
		})
	}

	snapshots, err := client.GetSecuritySnapshot(securityList)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(snapshots)
}

func handleSetReminder(client *gofutuapi.FutuClient, code string, rType, freq int, value float64, note string) {
	if code == "" {
		sendError(fmt.Errorf("code is required"))
		return
	}

	market, symbol, found := strings.Cut(code, ".")
	if !found {
		sendError(fmt.Errorf("invalid code format: %s", code))
		return
	}
	var m int32
	switch strings.ToUpper(market) {
	case "HK":
		m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
	case "US":
		m = int32(qotcommon.QotMarket_QotMarket_US_Security)
	default:
		sendError(fmt.Errorf("unsupported market for reminder: %s", market))
		return
	}

	security := &qotcommon.Security{
		Market: &m,
		Code:   &symbol,
	}

	key, err := client.SetPriceReminder(
		security,
		qotsetpricereminder.SetPriceReminderOp_SetPriceReminderOp_Add,
		qotcommon.PriceReminderType(rType),
		qotcommon.PriceReminderFreq(freq),
		value,
		note,
	)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(map[string]interface{}{
		"key":    key,
		"status": "added",
	})
}

func handleWatch(client *gofutuapi.FutuClient) {
	fmt.Println("Watching for real-time push notifications... (Ctrl+C to stop)")

	// Register order update handler
	client.OnPush(gofutuapi.TRD_UPDATEORDER, func(msg proto.Message) {
		resp := msg.(*trdupdateorder.Response)
		fmt.Print("\n[PUSH] Order Update:\n")
		json.NewEncoder(os.Stdout).Encode(resp.GetS2C())
	})

	// Keep alive until interrupted
	select {
	case <-client.Conn.Done():
		fmt.Println("Connection closed.")
	}
}

func handleFilter(client *gofutuapi.FutuClient, marketStr string, field int, min, max float64) {
	var m int32
	switch strings.ToUpper(marketStr) {
	case "HK":
		m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
	case "US":
		m = int32(qotcommon.QotMarket_QotMarket_US_Security)
	case "SH":
		m = int32(qotcommon.QotMarket_QotMarket_CNSH_Security)
	case "SZ":
		m = int32(qotcommon.QotMarket_QotMarket_CNSZ_Security)
	default:
		sendError(fmt.Errorf("unsupported market for filter: %s", marketStr))
		return
	}

	filters := &qotstockfilter.C2S{
		BaseFilterList: []*qotstockfilter.BaseFilter{
			{
				FieldName: proto.Int32(int32(field)),
				FilterMin: proto.Float64(min),
				FilterMax: proto.Float64(max),
			},
		},
	}

	result, err := client.StockFilter(qotcommon.QotMarket(m), filters)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(result)
}

func handleHistoryKL(client *gofutuapi.FutuClient, code string, begin, end string, klTypeStr, rehabTypeStr string) {
	if code == "" {
		sendError(fmt.Errorf("code is required"))
		return
	}

	market, symbol, found := strings.Cut(code, ".")
	if !found {
		sendError(fmt.Errorf("invalid code format: %s", code))
		return
	}
	var m int32
	switch strings.ToUpper(market) {
	case "HK":
		m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
	case "US":
		m = int32(qotcommon.QotMarket_QotMarket_US_Security)
	case "SH":
		m = int32(qotcommon.QotMarket_QotMarket_CNSH_Security)
	case "SZ":
		m = int32(qotcommon.QotMarket_QotMarket_CNSZ_Security)
	default:
		sendError(fmt.Errorf("unsupported market for history-kl: %s", market))
		return
	}

	security := &qotcommon.Security{
		Market: &m,
		Code:   &symbol,
	}

	klList, err := client.RequestHistoryKL(
		security,
		parseKLType(klTypeStr),
		parseRehabType(rehabTypeStr),
		begin,
		end,
		0,
	)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(klList)
}

func handleOptionExpiration(client *gofutuapi.FutuClient, code string) {
	if code == "" {
		sendError(fmt.Errorf("code is required"))
		return
	}

	market, symbol, found := strings.Cut(code, ".")
	if !found {
		sendError(fmt.Errorf("invalid code format: %s", code))
		return
	}
	var m int32
	switch strings.ToUpper(market) {
	case "HK":
		m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
	case "US":
		m = int32(qotcommon.QotMarket_QotMarket_US_Security)
	default:
		sendError(fmt.Errorf("unsupported market for options: %s", market))
		return
	}

	security := &qotcommon.Security{
		Market: &m,
		Code:   &symbol,
	}

	dates, err := client.GetOptionExpirationDate(security)
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(dates)
}

func handleOptionChain(client *gofutuapi.FutuClient, code string, begin, end string, optTypeStr string) {
	if code == "" {
		sendError(fmt.Errorf("code is required"))
		return
	}

	market, symbol, found := strings.Cut(code, ".")
	if !found {
		sendError(fmt.Errorf("invalid code format: %s", code))
		return
	}
	var m int32
	switch strings.ToUpper(market) {
	case "HK":
		m = int32(qotcommon.QotMarket_QotMarket_HK_Security)
	case "US":
		m = int32(qotcommon.QotMarket_QotMarket_US_Security)
	default:
		sendError(fmt.Errorf("unsupported market for options: %s", market))
		return
	}

	security := &qotcommon.Security{
		Market: &m,
		Code:   &symbol,
	}

	chain, err := client.GetOptionChain(security, begin, end, parseOptionType(optTypeStr))
	if err != nil {
		sendError(err)
		return
	}

	sendSuccess(chain)
}

func sendSuccess(data interface{}) {
	json.NewEncoder(os.Stdout).Encode(Response{
		Success: true,
		Data:    data,
	})
}

func sendError(err error) {
	json.NewEncoder(os.Stdout).Encode(Response{
		Success: false,
		Error:   err.Error(),
	})
}
