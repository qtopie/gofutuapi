from futu import *
import time

# 配置
ACC_ID = "acc_id"
CODE = 'US.MSFT'
PRICE = 429.8
QTY = 30
ENV = TrdEnv.REAL

trd_ctx = OpenSecTradeContext(filter_trdmarket=TrdMarket.US, host='127.0.0.1', port=11111, security_firm=SecurityFirm.FUTUSECURITIES)

# 打印账户确认
print(f"正在为账户 {ACC_ID} 执行卖单...")
print(f"标的: {CODE}, 数量: {QTY}, 价格: {PRICE}, 类型: 限价单 (GTC)")

# 下单
ret, data = trd_ctx.place_order(
    price=PRICE,
    qty=QTY,
    code=CODE,
    trd_side=TrdSide.SELL,
    order_type=OrderType.NORMAL,
    trd_env=ENV,
    acc_id=ACC_ID,
    time_in_force=TimeInForce.GTC
)

if ret == RET_OK:
    print("【下单成功】订单详情:")
    print(data)
else:
    print(f"【下单失败】: {data}")

trd_ctx.close()
