package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"c48x-airbridge/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewCommand(cli.Streams{Out: os.Stdout, Err: os.Stderr})
	if err := cmd.ExecuteContext(ctx); err != nil {
		slog.Error("command failed", slog.Any("err", err))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
