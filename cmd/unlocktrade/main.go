package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	trdcommon "github.com/qtopie/gofutuapi/gen/trade/common"
)

func main() {
	password := os.Getenv("FUTU_TRADE_PASSWORD")
	if password == "" {
		log.Fatal("FUTU_TRADE_PASSWORD is required")
	}

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
	if err := client.UnlockTrade(password, trdcommon.SecurityFirm_SecurityFirm_FutuSecurities); err != nil {
		log.Fatalf("failed to unlock trade: %v", err)
	}

	log.Println("trade unlocked successfully")
}
