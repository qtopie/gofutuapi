# Go Futu API Support Status

This document tracks the implementation status of the Futu OpenAPI features in the `gofutuapi` Go implementation.

## 1. Quote API (行情)

### Supported (已支持)
- **Subscription**: Subscribe, Unsubscribe, Query Subscription.
- **K-Line**: Real-time K-Line (`GetKLine`), History K-Line (`RequestHistoryKL`), Rehab factors.
- **Snapshots**: Market Snapshot (`GetSecuritySnapshot`).
- **Options**: Option Chain, Option Expiration Date.
- **Selection/Groups**: Stock Filter, User Security Groups, User Security List, Modify User Security.
- **Misc**: Price Reminders, Rehab.

### Missing in Client Wrapper (待封装)
- **Real-time Data**: 
  - [ ] Ticker (逐笔成交)
  - [ ] Order Book (买卖盘)
  - [ ] RT Data (分时数据)
  - [ ] Broker Queue (经纪队列 - HK only)
- **Market Info**:
  - [ ] Market State (市场状态)
  - [ ] Trade Date (交易日历)
  - [ ] Static Info (股票静态信息)
- **Plates & Categorization**:
  - [ ] Plate List (板块列表)
  - [ ] Plate Security (板块成分股)
  - [ ] Owner Plate (股票所属板块)
- **Analysis**:
  - [ ] Capital Flow (资金流向)
  - [ ] Capital Distribution (资金分布)
- **Derivatives**:
  - [ ] Reference Stock (关联股票查询)
  - [ ] Warrant/Bull-Bear (窝轮/牛熊证筛选)

## 2. Trade API (交易)

### Supported (已支持)
- **Accounts**: Account List (including Universal Accounts), Get Funds, Get Positions.
- **Orders**: Place Order, Modify Order, Cancel Order.
- **Queries**: Today's Order List, History Order List, Today's Order Fill List.
- **Security**: Unlock Trade.
- **User**: Get User Info.

### Missing in Client Wrapper (待封装)
- **Advanced Queries**:
  - [ ] Order Fee (订单费用查询)
  - [ ] Margin Ratio (融资融券比率)
  - [ ] Cash Flow (现金流水)
  - [ ] Max Trading Quantities (最大可交易数量)
  - [ ] History Order Fill List (历史成交详情)
- **Futures**:
  - [ ] Future-specific context and order placement logic.
- **Notifications**:
  - [ ] Integrated Trade Order/Deal push handlers.

## 3. System & Tools (系统工具)

### Missing in Client Wrapper (待封装)
- [ ] Global State (全局状态)
- [ ] Delay Statistics (延迟统计)

---

## Implementation Roadmap (实现路线图)

1. **Phase 1: Market Data Depth**: Ticker, OrderBook, RTData.
2. **Phase 2: Market Context**: Market State, Trade Date, Plate info.
3. **Phase 3: Advanced Trading Queries**: Order Fee, MaxTrdQtys, Margin Ratio.
4. **Phase 4: Specialized Data**: Capital Flow, Warrant.
5. **Phase 5: Futures & Push Handlers**.
