package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChargePi/chargeflow/cmd"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	)
	defer cancel()

	if err := cmd.Execute(ctx); err != nil {
		zap.L().Fatal("Unable to run", zap.Error(err))
	}

}
