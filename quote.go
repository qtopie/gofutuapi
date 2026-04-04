package gofutuapi

import (
	"fmt"

	"github.com/qtopie/gofutuapi/gen/qot/common"
	"github.com/qtopie/gofutuapi/gen/qot/getoptionexpirationdate"
	"github.com/qtopie/gofutuapi/gen/qot/qotgetkl"
	"github.com/qtopie/gofutuapi/gen/qot/qotgetoptionchain"
	"github.com/qtopie/gofutuapi/gen/qot/qotgetsecuritysnapshot"
	"github.com/qtopie/gofutuapi/gen/qot/qotrequesthistorykl"
	"github.com/qtopie/gofutuapi/gen/qot/qotrequestrehab"
	"github.com/qtopie/gofutuapi/gen/qot/qotsetpricereminder"
	"github.com/qtopie/gofutuapi/gen/qot/qotstockfilter"
	"github.com/qtopie/gofutuapi/gen/qot/qotsub"
	"google.golang.org/protobuf/proto"
)

// --- Real-Time Data ---

func (c *FutuClient) Subscribe(securityList []*common.Security, subTypeList []int32, isSub bool, isRegPush bool) error {
	if c == nil || c.Conn == nil {
		return fmt.Errorf("futu client connection is nil")
	}

	req := qotsub.Request{
		C2S: &qotsub.C2S{
			SecurityList:     securityList,
			SubTypeList:      subTypeList,
			IsSubOrUnSub:     &isSub,
			IsRegOrUnRegPush: &isRegPush,
		},
	}
	c.Conn.SendProto(QOT_SUB, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	var resp qotsub.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return fmt.Errorf("subscribe unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return fmt.Errorf("subscribe failed: %s", resp.GetRetMsg())
	}

	return nil
}

// --- Basic Data ---

func (c *FutuClient) GetKLine(security *common.Security, klType common.KLType, rehabType common.RehabType, count int32) ([]*common.KLine, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	k := int32(klType)
	r := int32(rehabType)
	req := qotgetkl.Request{
		C2S: &qotgetkl.C2S{
			Security:  security,
			KlType:    &k,
			RehabType: &r,
			ReqNum:    &count,
		},
	}
	c.Conn.SendProto(QOT_GETKL, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get kline failed: %w", err)
	}

	var resp qotgetkl.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get kline unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get kline failed: %s", resp.GetRetMsg())
	}

	return resp.GetS2C().GetKlList(), nil
}

func (c *FutuClient) GetSecuritySnapshot(securityList []*common.Security) ([]*qotgetsecuritysnapshot.Snapshot, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	req := qotgetsecuritysnapshot.Request{
		C2S: &qotgetsecuritysnapshot.C2S{
			SecurityList: securityList,
		},
	}
	c.Conn.SendProto(QOT_GETSECURITYSNAPSHOT, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get security snapshot failed: %w", err)
	}

	var resp qotgetsecuritysnapshot.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get security snapshot unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get security snapshot failed: %s", resp.GetRetMsg())
	}

	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetSnapshotList(), nil
}

func (c *FutuClient) SetPriceReminder(security *common.Security, op qotsetpricereminder.SetPriceReminderOp, reminderType common.PriceReminderType, freq common.PriceReminderFreq, value float64, note string) (int64, error) {
	if c == nil || c.Conn == nil {
		return 0, fmt.Errorf("futu client connection is nil")
	}

	o := int32(op)
	t := int32(reminderType)
	f := int32(freq)
	req := qotsetpricereminder.Request{
		C2S: &qotsetpricereminder.C2S{
			Security: security,
			Op:       &o,
			Type:     &t,
			Freq:     &f,
			Value:    &value,
			Note:     &note,
		},
	}
	c.Conn.SendProto(QOT_SETPRICEREMINDER, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return 0, fmt.Errorf("set price reminder failed: %w", err)
	}

	var resp qotsetpricereminder.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return 0, fmt.Errorf("set price reminder unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return 0, fmt.Errorf("set price reminder failed: %s", resp.GetRetMsg())
	}

	if resp.GetS2C() == nil {
		return 0, nil
	}

	return resp.GetS2C().GetKey(), nil
}

// --- Market Filter ---

func (c *FutuClient) StockFilter(market common.QotMarket, filters *qotstockfilter.C2S) (*qotstockfilter.S2C, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	m := int32(market)
	filters.Market = &m
	if filters.Begin == nil {
		filters.Begin = proto.Int32(0)
	}
	if filters.Num == nil {
		filters.Num = proto.Int32(50)
	}

	req := qotstockfilter.Request{
		C2S: filters,
	}
	c.Conn.SendProto(QOT_STOCKFILTER, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("stock filter failed: %w", err)
	}

	var resp qotstockfilter.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("stock filter unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("stock filter failed: %s", resp.GetRetMsg())
	}

	return resp.GetS2C(), nil
}

// --- History Data ---

func (c *FutuClient) RequestHistoryKL(security *common.Security, klType common.KLType, rehabType common.RehabType, begin, end string, maxNum int32) ([]*common.KLine, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	var allKL []*common.KLine
	var nextKey []byte

	for {
		k := int32(klType)
		r := int32(rehabType)
		req := qotrequesthistorykl.Request{
			C2S: &qotrequesthistorykl.C2S{
				Security:   security,
				KlType:     &k,
				RehabType:  &r,
				BeginTime:  &begin,
				EndTime:    &end,
				NextReqKey: nextKey,
			},
		}
		if maxNum > 0 {
			req.C2S.MaxAckKLNum = &maxNum
		}

		c.Conn.SendProto(QOT_REQUESTHISTORYKL, &req)

		reply, err := c.Conn.NextReplyPacket()
		if err != nil {
			return nil, fmt.Errorf("request history kl failed: %w", err)
		}

		var resp qotrequesthistorykl.Response
		if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
			return nil, fmt.Errorf("request history kl unmarshal failed: %w", err)
		}
		if resp.GetRetType() != 0 {
			return nil, fmt.Errorf("request history kl failed: %s", resp.GetRetMsg())
		}

		s2c := resp.GetS2C()
		if s2c == nil {
			break
		}
		allKL = append(allKL, s2c.GetKlList()...)
		nextKey = s2c.GetNextReqKey()

		if len(nextKey) == 0 {
			break
		}

		if maxNum > 0 && int32(len(allKL)) >= maxNum {
			break
		}
	}

	return allKL, nil
}

func (c *FutuClient) RequestRehab(security *common.Security) ([]*common.Rehab, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	req := qotrequestrehab.Request{
		C2S: &qotrequestrehab.C2S{
			Security: security,
		},
	}
	c.Conn.SendProto(QOT_REQUESTREHAB, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("request rehab failed: %w", err)
	}

	var resp qotrequestrehab.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("request rehab unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("request rehab failed: %s", resp.GetRetMsg())
	}

	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetRehabList(), nil
}

// --- Related Derivatives ---

func (c *FutuClient) GetOptionExpirationDate(owner *common.Security) ([]*getoptionexpirationdate.OptionExpirationDate, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	req := getoptionexpirationdate.Request{
		C2S: &getoptionexpirationdate.C2S{
			Owner: owner,
		},
	}
	c.Conn.SendProto(QOT_GETOPTIONEXPIRATIONDATE, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get option expiration date failed: %w", err)
	}

	var resp getoptionexpirationdate.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get option expiration date unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get option expiration date failed: %s", resp.GetRetMsg())
	}

	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetDateList(), nil
}

func (c *FutuClient) GetOptionChain(owner *common.Security, beginTime, endTime string, optionType common.OptionType) ([]*qotgetoptionchain.OptionChain, error) {
	if c == nil || c.Conn == nil {
		return nil, fmt.Errorf("futu client connection is nil")
	}

	t := int32(optionType)
	req := qotgetoptionchain.Request{
		C2S: &qotgetoptionchain.C2S{
			Owner:     owner,
			BeginTime: &beginTime,
			EndTime:   &endTime,
			Type:      &t,
		},
	}
	c.Conn.SendProto(QOT_GETOPTIONCHAIN, &req)

	reply, err := c.Conn.NextReplyPacket()
	if err != nil {
		return nil, fmt.Errorf("get option chain failed: %w", err)
	}

	var resp qotgetoptionchain.Response
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		return nil, fmt.Errorf("get option chain unmarshal failed: %w", err)
	}
	if resp.GetRetType() != 0 {
		return nil, fmt.Errorf("get option chain failed: %s", resp.GetRetMsg())
	}

	if resp.GetS2C() == nil {
		return nil, nil
	}

	return resp.GetS2C().GetOptionChain(), nil
}
