package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/qtopie/gofutuapi"
	qotcommon "github.com/qtopie/gofutuapi/gen/qot/common"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
)

//go:embed static/*
var staticFS embed.FS

var client *gofutuapi.FutuClient

type PositionItem struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Qty     float64 `json:"qty"`
	Price   float64 `json:"price"`
	Cost    float64 `json:"cost"`
	Val     float64 `json:"val"`
	PLVal   float64 `json:"plVal"`
	PLRatio float64 `json:"plRatio"`
}

func parseCode(codeStr string) (*qotcommon.Security, error) {
	parts := strings.SplitN(codeStr, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid code format, expected MARKET.CODE: %s", codeStr)
	}
	marketMap := map[string]int32{
		"HK": int32(qotcommon.QotMarket_QotMarket_HK_Security),
		"US": int32(qotcommon.QotMarket_QotMarket_US_Security),
		"SH": int32(qotcommon.QotMarket_QotMarket_CNSH_Security),
		"SZ": int32(qotcommon.QotMarket_QotMarket_CNSZ_Security),
	}
	market, ok := marketMap[strings.ToUpper(parts[0])]
	if !ok {
		return nil, fmt.Errorf("unknown market: %s", parts[0])
	}
	return &qotcommon.Security{
		Market: &market,
		Code:   &parts[1],
	}, nil
}

func initClient() error {
	ctx := context.Background()
	conn, err := gofutuapi.OpenContext(ctx, gofutuapi.FutuApiOption{
		Address: "127.0.0.1:11111",
	})
	if err != nil {
		return err
	}
	client = gofutuapi.NewClient(conn)
	return nil
}

func apiPortfolio(w http.ResponseWriter, r *http.Request) {
	if client == nil {
		http.Error(w, "Not connected", 500)
		return
	}
	acc, err := client.FindTradeAccount(trdcommon.TrdEnv_TrdEnv_Real, trdcommon.TrdMarket_TrdMarket_HK)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	positions, err := client.GetPositionsForAccount(acc, true)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var res []PositionItem
	for _, p := range positions {
		if p.GetQty() == 0 {
			continue
		}
		marketStr := "HK"
		if p.GetSecMarket() == int32(qotcommon.QotMarket_QotMarket_US_Security) {
			marketStr = "US"
		} else if p.GetSecMarket() == int32(qotcommon.QotMarket_QotMarket_CNSH_Security) {
			marketStr = "SH"
		} else if p.GetSecMarket() == int32(qotcommon.QotMarket_QotMarket_CNSZ_Security) {
			marketStr = "SZ"
		}

		res = append(res, PositionItem{
			Code:    marketStr + "." + p.GetCode(),
			Name:    p.GetName(),
			Qty:     p.GetQty(),
			Price:   p.GetPrice(),
			Cost:    p.GetCostPrice(),
			Val:     p.GetVal(),
			PLVal:   p.GetPlVal(),
			PLRatio: p.GetPlRatio(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func apiSnapshot(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Query().Get("code")
	sec, err := parseCode(codeStr)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	snaps, err := client.GetSecuritySnapshot([]*qotcommon.Security{sec})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(snaps) == 0 {
		http.Error(w, "no snapshot data", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snaps[0])
}

func apiKline(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Query().Get("code")
	sec, err := parseCode(codeStr)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	beginTime := time.Now().AddDate(0, -1, -15).Format("2006-01-02") // 45 days ago to get ~30 trading days
	endTime := time.Now().Format("2006-01-02")

	klines, err := client.RequestHistoryKL(sec, qotcommon.KLType_KLType_Day, qotcommon.RehabType_RehabType_Forward, beginTime, endTime, 100)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type KLPoint struct {
		Time   string  `json:"time"`
		Open   float64 `json:"open"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Close  float64 `json:"close"`
		Volume int64   `json:"volume"`
	}
	var res []KLPoint
	for _, k := range klines {
		if k.GetIsBlank() {
			continue
		}
		res = append(res, KLPoint{
			Time:   k.GetTime()[:10], // YYYY-MM-DD
			Open:   k.GetOpenPrice(),
			High:   k.GetHighPrice(),
			Low:    k.GetLowPrice(),
			Close:  k.GetClosePrice(),
			Volume: k.GetVolume(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func main() {
	if err := initClient(); err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}
	defer client.Conn.Close()

	http.HandleFunc("/api/portfolio", apiPortfolio)
	http.HandleFunc("/api/snapshot", apiSnapshot)
	http.HandleFunc("/api/kline", apiKline)
	http.Handle("/", http.FileServer(http.FS(staticFS)))

	fmt.Println("Dashboard started at http://localhost:8080/static/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
