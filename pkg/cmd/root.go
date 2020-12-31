package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cretz/takecast/pkg/receiver"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use: "takecast",
	}
	cmd.PersistentFlags().StringP("cert-dir", "d", ".", "Dir to load/create ca.crt and ca.key")
	cmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error, or off)")
	cmd.AddCommand(patchCmd(), recordCmd(), unpatchCmd())
	return cmd
}

type rootContext struct {
	context.Context
	cmd     *cobra.Command
	log     receiver.Log
	certDir string
}

func applyRun(cmd *cobra.Command, fn func(*rootContext) error) *cobra.Command {
	if cmd.Args == nil {
		cmd.Args = cobra.NoArgs
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := &rootContext{cmd: cmd}
		var cancel context.CancelFunc
		ctx.Context, cancel = context.WithCancel(context.Background())
		defer cancel()
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		var err error
		if ctx.log, err = receiver.NewStdLog(logLevel, nil); err != nil {
			log.Fatalf("failed creating log: %v", err)
			return
		}
		ctx.certDir, _ = cmd.Root().PersistentFlags().GetString("cert-dir")
		// Run in background
		errCh := make(chan error, 1)
		go func() { errCh <- fn(ctx) }()
		// Wait for error or termination
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		select {
		case err := <-errCh:
			if err != nil {
				log.Fatal(err)
			}
		case <-sigCh:
			ctx.log.Infof("Got termination signal, closing")
		}
	}
	return cmd
}
