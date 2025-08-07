package gofutuapi

import (
	"crypto/sha1"
	"encoding/binary"
)

const (
	HEADER_SIZE  = 2 + 4 + 1 + 1 + 4 + 4 + 20 + 8
	szHeaderFlag = "FT"

	// 初始化连接
	INIT_CONNECT = 1001
	// 获取全局状态
	GETGLOBALSTATE = 1002
	// 推送通知
	NOTIFY = 1003
	// 心跳
	KEEP_ALIVE = 1004

	// 订阅或者反订阅
	QOT_SUB = 3001
	// 注册推送
	QOT_REGQOTPUSH = 3002
	// 获取订阅信息
	QOT_GETSUBINFO = 3003
	// 获取逐笔,调用该接口前需要先订阅(订阅位：Qot_Common.SubType_Ticker)
	QOT_GETTICKER = 3010
	// 获取基本行情,调用该接口前需要先订阅(订阅位：Qot_Common.SubType_Basic)
	QOT_GETBASICQOT = 3004
	// 获取摆盘,调用该接口前需要先订阅(订阅位：Qot_Common.SubType_OrderBook)
	QOT_GETORDERBOOK = 3012
	// 获取K线，调用该接口前需要先订阅(订阅位：Qot_Common.SubType_KL_XXX)
	QOT_GETKL = 3006
	// 获取分时，调用该接口前需要先订阅(订阅位：Qot_Common.SubType_RT)
	QOT_GETRT = 3008
	// 获取经纪队列，调用该接口前需要先订阅(订阅位：Qot_Common.SubType_Broker)
	QOT_GETBROKER = 3014
	// 获取本地历史复权信息
	QOT_GETREHAB = 3102
	// 在线请求历史复权信息，不读本地历史数据DB
	QOT_REQUESTREHAB = 3105
	// 在线请求历史K线，不读本地历史数据DB
	QOT_REQUESTHISTORYKL = 3103
	// 获取历史K线已经用掉的额度
	QOT_REQUESTHISTORYKLQUOTA = 3104
	// 获取交易日
	QOT_GETTRADEDATE = 3200
	// 获取静态信息
	QOT_GETSTATICINFO = 3202
	// 获取股票快照
	QOT_GETSECURITYSNAPSHOT = 3203
	// 获取板块集合下的板块
	QOT_GETPLATESET = 3204
	// 获取板块下的股票
	QOT_GETPLATESECURITY = 3205
	// 获取相关股票
	QOT_GETREFERENCE = 3206
	// 获取股票所属的板块
	QOT_GETOWNERPLATE = 3207
	// 获取大股东持股变化列表
	QOT_GETHOLDINGCHANGELIST = 3208
	// 筛选期权
	QOT_GETOPTIONCHAIN = 3209
	// 筛选窝轮
	QOT_GETWARRANT = 3210
	// 获取资金流向
	QOT_GETCAPITALFLOW = 3211
	// 获取资金分布
	QOT_GETCAPITALDISTRIBUTION = 3212
	// 获取自选股分组下的股票
	QOT_GETUSERSECURITY = 3213
	// 修改自选股分组下的股票
	QOT_MODIFYUSERSECURITY = 3214
	// 推送基本行情
	QOT_UPDATEBASICQOT = 3005
	// 推送K线
	QOT_UPDATEKL = 3007
	// 推送分时
	QOT_UPDATERT = 3009
	// 推送逐笔
	QOT_UPDATETICKER = 3011
	// 推送买卖盘
	QOT_UPDATEORDERBOOK = 3013
	// 推送经纪队列
	QOT_UPDATEBROKER = 3015
	// 到价提醒通知
	QOT_UPDATEPRICEREMINDER = 3019
	// 获取条件选股
	QOT_STOCKFILTER = 3215
	// 获取股票代码变化信息
	QOT_GETCODECHANGE = 3216
	// 获取新股Ipo
	QOT_GETIPOLIST = 3217
	// 获取期货合约资料
	QOT_GETFUTUREINFO = 3218
	// 在线拉取交易日
	QOT_REQUESTTRADEDATE = 3219
	// 设置到价提醒
	QOT_SETPRICEREMINDER = 3220
	// 获取到价提醒
	QOT_GETPRICEREMINDER = 3221
	// 获取自选股分组
	QOT_GETUSERSECURITYGROUP = 3222
	// 获取指定品种的市场状态
	QOT_GETMARKETSTATE = 3223
	// 获取期权到期日
	QOT_GETOPTIONEXPIRATIONDATE = 3224

	// 获取交易账户列表
	TRD_GETACCLIST = 2001
	// 解锁
	TRD_UNLOCKTRADE = 2005
	// 订阅接收推送数据的交易账户
	TRD_SUBACCPUSH = 2008
	// 获取账户资金
	TRD_GETFUNDS = 2101
	// 获取账户持仓
	TRD_GETPOSITIONLIST = 2102
	// 获取最大交易数量
	TRD_GETMAXTRDQTYS = 2111
	// 获取当日订单列表
	TRD_GETORDERLIST = 2201
	// 下单
	TRD_PLACEORDER = 2202
	// 修改订单
	TRD_MODIFYORDER = 2205
	// 订单状态变动通知(推送)
	TRD_UPDATEORDER = 2208
	// 获取当日成交列表
	TRD_GETORDERFILLLIST = 2211
	// 成交通知(推送)
	TRD_UPDATEORDERFILL = 2218
	// 获取历史订单列表
	TRD_GETHISTORYORDERLIST = 2221
	// 获取历史成交列表
	TRD_GETHISTORYORDERFILLLIST = 2222
	// 获取融资融券数据
	TRD_GETMARGINRATIO = 2223
	// 获取融资融券数据
	TRD_GETORDERFEE = 2225
	// 获取资金流水
	TRD_GETFLOWSUMMARY = 2226
)

// IsPushProto 检查给定的 protoID 是否为推送协议。
func IsPushProto(protoID int) bool {
	switch protoID {
	case QOT_UPDATEBASICQOT,
		QOT_UPDATEBROKER,
		QOT_UPDATEKL,
		QOT_UPDATEORDERBOOK,
		QOT_UPDATEPRICEREMINDER,
		QOT_UPDATERT,
		QOT_UPDATETICKER,
		TRD_UPDATEORDER,
		TRD_UPDATEORDERFILL,
		NOTIFY:
		return true
	default:
		return false
	}
}


type ProtoHeader struct {
	szHeaderFlag [2]byte
	ProtoID      int32
	ProtoFmtType byte
	ProtoVer     byte
	SerialNo     int32
	BodyLen      int32
	arrBodySHA1  [20]byte
	arrReserved  [8]byte
}

func NewHeader() *ProtoHeader {
	header := &ProtoHeader{}

	header.arrReserved = [8]byte{}
	return header
}

func ParseHeader(data []byte) *ProtoHeader {
	if len(data) != HEADER_SIZE {
		panic("unmatched header size")
	}

	header := NewHeader()

	header.ProtoID = bytesToInt32(data[2:6])
	header.ProtoFmtType = data[6]
	header.ProtoVer = data[7]
	header.SerialNo = bytesToInt32(data[8:12])
	header.BodyLen = bytesToInt32(data[12:16])
	copy(header.arrBodySHA1[:], data[16:36])
	copy(header.arrReserved[:], data[36:])
	return header
}

func (h *ProtoHeader) UpdateBodyInfo(b []byte) {
	h.BodyLen = int32(len(b))
	h.arrBodySHA1 = sha1.Sum(b)
}

func (h *ProtoHeader) ToBytes() []byte {
	data := make([]byte, HEADER_SIZE)
	copy(data, szHeaderFlag)
	copy(data[2:6], int32ToBytes(h.ProtoID))
	data[6] = h.ProtoFmtType
	data[7] = h.ProtoVer
	copy(data[8:12], int32ToBytes(h.SerialNo))
	copy(data[12:16], int32ToBytes(h.BodyLen))
	copy(data[16:36], h.arrBodySHA1[:])
	copy(data[36:], h.arrReserved[:])
	return data
}

func int32ToBytes(n int32) []byte {
	b := make([]byte, 4)

	binary.LittleEndian.PutUint32(b, uint32(n))

	return b
}

func bytesToInt32(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b[:]))
}
