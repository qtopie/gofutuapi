package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
)

func parseMarket(value string) (trdcommon.TrdMarket, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "US":
		return trdcommon.TrdMarket_TrdMarket_US, nil
	case "HK":
		return trdcommon.TrdMarket_TrdMarket_HK, nil
	default:
		return 0, fmt.Errorf("unsupported market %q, expected US or HK", value)
	}
}

func parseEnv(value string) (trdcommon.TrdEnv, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "REAL":
		return trdcommon.TrdEnv_TrdEnv_Real, nil
	case "SIMULATE", "SIM":
		return trdcommon.TrdEnv_TrdEnv_Simulate, nil
	default:
		return 0, fmt.Errorf("unsupported env %q, expected REAL or SIMULATE", value)
	}
}

func printAccount(label string, account *trdcommon.TrdAcc) {
	fmt.Printf("%s account info:\n", label)
	fmt.Printf("  accID=%d\n", account.GetAccID())
	fmt.Printf("  trdEnv=%s\n", trdcommon.TrdEnv(account.GetTrdEnv()).String())
	fmt.Printf("  trdMarketAuthList=%v\n", account.GetTrdMarketAuthList())
	fmt.Printf("  accType=%s\n", trdcommon.TrdAccType(account.GetAccType()).String())
	fmt.Printf("  cardNum=%s\n", account.GetCardNum())
	fmt.Printf("  securityFirm=%s\n", trdcommon.SecurityFirm(account.GetSecurityFirm()).String())
	fmt.Printf("  simAccType=%s\n", trdcommon.SimAccType(account.GetSimAccType()).String())
	fmt.Printf("  accStatus=%d\n", account.GetAccStatus())
	fmt.Printf("  uniCardNum=%s\n", account.GetUniCardNum())
}

func currencyLabel(currency int32) string {
	if name, ok := trdcommon.Currency_name[currency]; ok {
		return name
	}
	return fmt.Sprintf("%d", currency)
}

func printFunds(label string, account *trdcommon.TrdAcc, funds *trdcommon.Funds) {
	if funds == nil {
		fmt.Printf("%s account %d funds: <nil>\n", label, account.GetAccID())
		return
	}
	fmt.Printf("%s account %d funds:\n", label, account.GetAccID())
	fmt.Printf("  currency=%s\n", currencyLabel(funds.GetCurrency()))
	fmt.Printf("  cash=%.2f\n", funds.GetCash())
	fmt.Printf("  frozenCash=%.2f\n", funds.GetFrozenCash())
	fmt.Printf("  availableFunds=%.2f\n", funds.GetAvailableFunds())
	fmt.Printf("  netCashPower=%.2f\n", funds.GetNetCashPower())
	fmt.Printf("  power=%.2f\n", funds.GetPower())
	fmt.Printf("  totalAssets=%.2f\n", funds.GetTotalAssets())
}

func printPositions(label string, account *trdcommon.TrdAcc, positions []*trdcommon.Position) {
	fmt.Printf("%s account %d holdings: %d\n", label, account.GetAccID(), len(positions))
	for i, position := range positions {
		fmt.Printf("%d. code=%s name=%s qty=%.2f price=%.2f cost=%.2f value=%.2f\n",
			i+1,
			position.GetCode(),
			position.GetName(),
			position.GetQty(),
			position.GetPrice(),
			position.GetAverageCostPrice(),
			position.GetVal(),
		)
	}
}

func printOrders(label string, account *trdcommon.TrdAcc, orders []*trdcommon.Order) {
	fmt.Printf("%s account %d orders: %d\n", label, account.GetAccID(), len(orders))
	for i, order := range orders {
		fmt.Printf("%d. code=%s name=%s qty=%.2f price=%.2f status=%s created=%s\n",
			i+1,
			order.GetCode(),
			order.GetName(),
			order.GetQty(),
			order.GetPrice(),
			gofutuapi.OrderStatusLabel(order.GetOrderStatus()),
			order.GetCreateTime(),
		)
	}
}

func findAccountByID(accounts []*trdcommon.TrdAcc, accID uint64) *trdcommon.TrdAcc {
	for _, account := range accounts {
		if account.GetAccID() == accID {
			return account
		}
	}
	return nil
}

func filterAccountsByEnv(accounts []*trdcommon.TrdAcc, trdEnv int32) []*trdcommon.TrdAcc {
	filtered := make([]*trdcommon.TrdAcc, 0, len(accounts))
	for _, account := range accounts {
		if account.GetTrdEnv() == trdEnv {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func main() {
	marketFlag := flag.String("market", "US", "market to query: US or HK")
	envFlag := flag.String("env", "REAL", "trading environment to query: REAL or SIMULATE")
	accIDFlag := flag.Uint64("acc-id", 0, "exact account ID to query")
	allAccountsFlag := flag.Bool("all-accounts", false, "print holdings and orders for every available account")
	debugAccountsFlag := flag.Bool("debug-accounts", false, "print all available trade accounts before selecting one")
	flag.Parse()

	market, err := parseMarket(*marketFlag)
	if err != nil {
		log.Fatal(err)
	}
	trdEnv, err := parseEnv(*envFlag)
	if err != nil {
		log.Fatal(err)
	}
	marketLabel := market.String()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := gofutuapi.NewClient(conn)
	// unlockFirm := trdcommon.SecurityFirm_SecurityFirm_FutuSecurities
	// if err := client.UnlockTrade(password, unlockFirm); err != nil {
	// 	log.Fatalf("failed to unlock trade: %v", err)
	// }
	// fmt.Printf("unlock trade: success (securityFirm=%s)\n", unlockFirm.String())
	accounts, err := client.GetTradeAccounts()
	if err != nil {
		log.Fatalf("failed to get trade accounts: %v", err)
	}
	accounts = filterAccountsByEnv(accounts, int32(trdEnv))
	if *debugAccountsFlag {
		fmt.Printf("available trade accounts: %d\n", len(accounts))
		for i, account := range accounts {
			printAccount(fmt.Sprintf("available[%d]", i+1), account)
		}
	}
	if *allAccountsFlag {
		for i, account := range accounts {
			label := fmt.Sprintf("account[%d]", i+1)
			printAccount(label, account)

			funds, err := client.GetFundsForAccount(account, true)
			if err != nil {
				fmt.Printf("%s funds error: %v\n", label, err)
			} else {
				printFunds(label, account, funds)
			}

			positions, err := client.GetPositionsForAccount(account, true)
			if err != nil {
				fmt.Printf("%s positions error: %v\n", label, err)
			} else {
				printPositions(label, account, positions)
			}

			orders, err := client.GetOrderListForAccount(account, nil, true)
			if err != nil {
				fmt.Printf("%s orders error: %v\n", label, err)
			} else {
				printOrders(label, account, orders)
			}
		}
		return
	}

	var account *trdcommon.TrdAcc
	if *accIDFlag != 0 {
		account = findAccountByID(accounts, *accIDFlag)
		if account == nil {
			log.Fatalf("failed to find account with accID=%d", *accIDFlag)
		}
		marketLabel = fmt.Sprintf("accID=%d", account.GetAccID())
	} else {
		account, err = client.FindTradeAccount(trdEnv, market)
		if err != nil {
			log.Fatalf("failed to find %s %s account: %v", trdEnv.String(), marketLabel, err)
		}
	}
	printAccount(marketLabel, account)

	funds, err := client.GetFundsForAccount(account, true)
	if err != nil {
		log.Fatalf("failed to get %s funds: %v", marketLabel, err)
	}
	positions, err := client.GetPositionsForAccount(account, true)
	if err != nil {
		log.Fatalf("failed to get %s positions: %v", marketLabel, err)
	}
	orders, err := client.GetOrderListForAccount(account, nil, true)
	if err != nil {
		log.Fatalf("failed to get %s orders: %v", marketLabel, err)
	}

	printFunds(marketLabel, account, funds)
	printPositions(marketLabel, account, positions)
	printOrders(marketLabel, account, orders)
}
