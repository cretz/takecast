package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cretz/takecast/pkg/cert"
	"github.com/cretz/takecast/pkg/receiver/mirror"
	"github.com/cretz/takecast/pkg/server"
	"github.com/spf13/cobra"
)

func recordCmd() *cobra.Command {
	var outDir string
	cmd := applyRun(
		&cobra.Command{
			Use:   "record",
			Short: "Record all incoming streams without transcoding",
		},
		func(ctx *rootContext) error {
			// Load root CA
			rootCA, err := cert.LoadKeyPairFromFiles(filepath.Join("ca.crt"), filepath.Join("ca.key"))
			if err != nil {
				return fmt.Errorf("failed loading ca.crt/ca.key, did you forget to run 'patch'? err: %w", err)
			}
			// Start server listen
			s, err := server.Listen(server.Config{RootCACert: rootCA, Log: ctx.log})
			if err != nil {
				return fmt.Errorf("failed starting server: %w", err)
			}
			defer s.Close()
			// Register mirror application
			if m, err := mirror.New(mirror.Config{Log: ctx.log}); err != nil {
				return fmt.Errorf("failed creating mirror application: %w", err)
			} else if err = s.Receiver.RegisterApplication(m); err != nil {
				return fmt.Errorf("failed registering mirror application: %w", err)
			}
			// // Register YouTube application
			// if y, err := youtube.New(youtube.Config{Log: ctx.log}); err != nil {
			// 	return fmt.Errorf("failed creating YouTube application: %w", err)
			// } else if err = s.Receiver.RegisterApplication(y); err != nil {
			// 	return fmt.Errorf("failed registering YouTube application: %w", err)
			// }
			// Run server in background
			errCh := make(chan error, 1)
			go func() { errCh <- s.Serve() }()
			// Wait for context done or error
			select {
			case <-ctx.Done():
				return nil
			case err := <-errCh:
				return fmt.Errorf("server failed: %w", err)
			}
		},
	)
	cmd.Flags().StringVarP(&outDir, "out-dir", "o", ".", "Directory to record videos in")
	return cmd
}
