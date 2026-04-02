package gofutuapi

import (
	"testing"

	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
)

func TestPendingOrderStatusesReturnsCopy(t *testing.T) {
	first := PendingOrderStatuses()
	second := PendingOrderStatuses()

	if len(first) == 0 {
		t.Fatal("PendingOrderStatuses returned an empty list")
	}

	first[0] = -999
	if second[0] == -999 {
		t.Fatal("PendingOrderStatuses should return a copy")
	}
}

func TestTradeHeaderFromAccountUsesFirstAuthorizedMarket(t *testing.T) {
	trdEnv := int32(trdcommon.TrdEnv_TrdEnv_Real)
	accID := uint64(42)
	acc := &trdcommon.TrdAcc{
		TrdEnv:            &trdEnv,
		AccID:             &accID,
		TrdMarketAuthList: []int32{int32(trdcommon.TrdMarket_TrdMarket_US), int32(trdcommon.TrdMarket_TrdMarket_HK)},
	}

	header := tradeHeaderFromAccount(acc)

	if header.GetTrdEnv() != trdEnv {
		t.Fatalf("unexpected trading environment: got %d want %d", header.GetTrdEnv(), trdEnv)
	}
	if header.GetAccID() != accID {
		t.Fatalf("unexpected account id: got %d want %d", header.GetAccID(), accID)
	}
	if header.GetTrdMarket() != int32(trdcommon.TrdMarket_TrdMarket_US) {
		t.Fatalf("unexpected trade market: got %d", header.GetTrdMarket())
	}
}

func TestFindTradeAccount(t *testing.T) {
	real := int32(trdcommon.TrdEnv_TrdEnv_Real)
	sim := int32(trdcommon.TrdEnv_TrdEnv_Simulate)
	accs := []*trdcommon.TrdAcc{
		{TrdEnv: &sim, TrdMarketAuthList: []int32{int32(trdcommon.TrdMarket_TrdMarket_US)}},
		{TrdEnv: &real, TrdMarketAuthList: []int32{int32(trdcommon.TrdMarket_TrdMarket_HK)}},
		{TrdEnv: &real, TrdMarketAuthList: []int32{int32(trdcommon.TrdMarket_TrdMarket_US)}},
	}

	acc := findTradeAccount(accs, real, int32(trdcommon.TrdMarket_TrdMarket_US))
	if acc == nil {
		t.Fatal("expected to find real US account")
	}
	if acc.GetTrdEnv() != real {
		t.Fatalf("unexpected trading environment: got %d want %d", acc.GetTrdEnv(), real)
	}
	if len(acc.GetTrdMarketAuthList()) == 0 || acc.GetTrdMarketAuthList()[0] != int32(trdcommon.TrdMarket_TrdMarket_US) {
		t.Fatalf("unexpected trade market list: got %v", acc.GetTrdMarketAuthList())
	}
}

func TestMD5Hex(t *testing.T) {
	if got := md5Hex("123456"); got != "e10adc3949ba59abbe56e057f20f883e" {
		t.Fatalf("unexpected md5 hex: got %s", got)
	}
}
