package gofutuapi

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetacclist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetfunds"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetorderlist"
	"github.com/qtopie/gofutuapi/gen/trade/trdgetpositionlist"
	"github.com/qtopie/gofutuapi/gen/trade/trdunlocktrade"
	"google.golang.org/protobuf/proto"
)

type FutuClient struct {
	Conn *FutuApiConn
}

func NewClient(conn *FutuApiConn) *FutuClient {
	return &FutuClient{Conn: conn}
}

var pendingOrderStatuses = []int32{
	int32(trdcommon.OrderStatus_OrderStatus_WaitingSubmit),
	int32(trdcommon.OrderStatus_OrderStatus_Submitting),
	int32(trdcommon.OrderStatus_OrderStatus_Submitted),
	int32(trdcommon.OrderStatus_OrderStatus_Filled_Part),
	int32(trdcommon.OrderStatus_OrderStatus_Cancelling_Part),
	int32(trdcommon.OrderStatus_OrderStatus_Cancelling_All),
	int32(trdcommon.OrderStatus_OrderStatus_TimeOut),
}

func PendingOrderStatuses() []int32 {
	statuses := make([]int32, len(pendingOrderStatuses))
	copy(statuses, pendingOrderStatuses)
	return statuses
}

func OrderStatusLabel(status int32) string {
	if name, ok := trdcommon.OrderStatus_name[status]; ok {
		return name
	}
	return fmt.Sprintf("%d", status)
}

func (c *FutuClient) UnlockTrade(password string, securityFirm trdcommon.SecurityFirm) error {
	if c == nil || c.Conn == nil {
		return fmt.Errorf("futu client connection is nil")
	}
	if password == "" {
		return fmt.Errorf("trade password is empty")
	}

	unlock := true
	pwdMD5 := md5Hex(password)
	firm := int32(securityFirm)
	req := trdunlocktrade.Request{
		C2S: &trdunlocktrade.C2S{
			Unlock:       &unlock,
			PwdMD5:       &pwdMD5,
			SecurityFirm: &firm,
		},
	}
	c.Conn.SendProto(TRD_UNLOCKTRADE, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return fmt.Errorf("unlock trade failed: %w", err)
	}

	var resp trdunlocktrade.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return fmt.Errorf("unlock trade unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return fmt.Errorf("unlock trade failed: %s", resp.GetRetMsg())
	}

	return nil
}

func (c *FutuClient) GetPendingOrders(trdEnv trdcommon.TrdEnv, trdMarket trdcommon.TrdMarket) ([]*trdcommon.Order, error) {
	return c.GetOrderList(trdEnv, trdMarket, PendingOrderStatuses(), true)
}

func (c *FutuClient) GetOrderList(trdEnv trdcommon.TrdEnv, trdMarket trdcommon.TrdMarket, filterStatusList []int32, refreshCache bool) ([]*trdcommon.Order, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	acc, err := c.FindTradeAccount(trdEnv, trdMarket)
	if err != nil {
		return nil, err
	}

	return c.GetOrderListForAccount(acc, filterStatusList, refreshCache)
}

func (c *FutuClient) GetOrderListForAccount(acc *trdcommon.TrdAcc, filterStatusList []int32, refreshCache bool) ([]*trdcommon.Order, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	header, err := c.tradeHeaderForAccount(acc)
	if err != nil {
		return nil, err
	}

	req := trdgetorderlist.Request{
		C2S: &trdgetorderlist.C2S{
			Header:           header,
			FilterStatusList: append([]int32(nil), filterStatusList...),
			RefreshCache:     &refreshCache,
		},
	}
	c.Conn.SendProto(TRD_GETORDERLIST, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get order list failed: %w", err)
	}

	var resp trdgetorderlist.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get order list unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get order list failed: %s", resp.GetRetMsg())
	}
	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetOrderList(), nil
}

func (c *FutuClient) GetPositionsForAccount(acc *trdcommon.TrdAcc, refreshCache bool) ([]*trdcommon.Position, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	header, err := c.tradeHeaderForAccount(acc)
	if err != nil {
		return nil, err
	}

	req := trdgetpositionlist.Request{
		C2S: &trdgetpositionlist.C2S{
			Header:       header,
			RefreshCache: &refreshCache,
		},
	}
	c.Conn.SendProto(TRD_GETPOSITIONLIST, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get position list failed: %w", err)
	}

	var resp trdgetpositionlist.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get position list unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get position list failed: %s", resp.GetRetMsg())
	}
	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetPositionList(), nil
}

func (c *FutuClient) GetFundsForAccount(acc *trdcommon.TrdAcc, refreshCache bool) (*trdcommon.Funds, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	header, err := c.tradeHeaderForAccount(acc)
	if err != nil {
		return nil, err
	}
	currency := int32(trdcommon.Currency_Currency_USD)

	req := trdgetfunds.Request{
		C2S: &trdgetfunds.C2S{
			Header:       header,
			RefreshCache: &refreshCache,
			Currency:     &currency,
		},
	}
	c.Conn.SendProto(TRD_GETFUNDS, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get funds failed: %w", err)
	}

	var resp trdgetfunds.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get funds unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get funds failed: %s", resp.GetRetMsg())
	}
	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetFunds(), nil
}

func (c *FutuClient) GetTradeAccounts() ([]*trdcommon.TrdAcc, error) {
	return c.getTradeAccounts(
		int32(trdcommon.TrdCategory_TrdCategory_Security),
		true,
	)
}

func (c *FutuClient) FindTradeAccount(trdEnv trdcommon.TrdEnv, trdMarket trdcommon.TrdMarket) (*trdcommon.TrdAcc, error) {
	accounts, err := c.GetTradeAccounts()
	if err != nil {
		return nil, err
	}

	acc := findTradeAccount(accounts, int32(trdEnv), int32(trdMarket))
	if acc == nil {
		return nil, fmt.Errorf("trading account not found for env=%s market=%s", trdEnv.String(), trdMarket.String())
	}

	return acc, nil
}

func (c *FutuClient) getTradeAccounts(trdCategory int32, needGeneralSecAccount bool) ([]*trdcommon.TrdAcc, error) {
	userID := uint64(0)
	req := trdgetacclist.Request{
		C2S: &trdgetacclist.C2S{
			UserID:                &userID,
			TrdCategory:           &trdCategory,
			NeedGeneralSecAccount: &needGeneralSecAccount,
		},
	}
	c.Conn.SendProto(TRD_GETACCLIST, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get acc list failed: %w", err)
	}

	var resp trdgetacclist.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get acc list unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get acc list failed: %s", resp.GetRetMsg())
	}

	s2c := resp.GetS2C()
	if s2c == nil || len(s2c.GetAccList()) == 0 {
		return nil, nil
	}

	return s2c.GetAccList(), nil
}

func (c *FutuClient) tradeHeaderForAccount(acc *trdcommon.TrdAcc) (*trdcommon.TrdHeader, error) {
	if acc == nil {
		return nil, fmt.Errorf("trading account is nil")
	}

	return tradeHeaderFromAccount(acc), nil
}

func tradeHeaderFromAccount(acc *trdcommon.TrdAcc) *trdcommon.TrdHeader {
	trdEnv := acc.GetTrdEnv()
	accID := acc.GetAccID()
	trdMarket := int32(trdcommon.TrdMarket_TrdMarket_Unknown)
	if len(acc.GetTrdMarketAuthList()) > 0 {
		trdMarket = acc.GetTrdMarketAuthList()[0]
	}

	return &trdcommon.TrdHeader{
		TrdEnv:    &trdEnv,
		AccID:     &accID,
		TrdMarket: &trdMarket,
	}
}

func findTradeAccount(accounts []*trdcommon.TrdAcc, trdEnv int32, trdMarket int32) *trdcommon.TrdAcc {
	for _, acc := range accounts {
		if acc.GetTrdEnv() != trdEnv {
			continue
		}
		for _, market := range acc.GetTrdMarketAuthList() {
			if market == trdMarket {
				return acc
			}
		}
	}
	return nil
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
