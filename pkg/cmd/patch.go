package cmd

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cretz/takecast/pkg/chrome"
	"github.com/spf13/cobra"
)

func patchCmd() *cobra.Command {
	return applyRun(
		&cobra.Command{
			Use:   "patch [path to chrome parent dir]",
			Short: "Patch Chrome for use with TakeCast",
			Args:  cobra.ExactArgs(1),
		},
		func(ctx *rootContext) error {
			certDirAbs, err := filepath.Abs(ctx.certDir)
			if err != nil {
				return fmt.Errorf("invalid cert dir: %w", err)
			}
			// Load existing root CA
			existingCA := chrome.LoadExistingRootCADERBytes()
			// Grab or create bytes to replace
			certFile := filepath.Join(certDirAbs, "ca.crt")
			certBytes, err := ioutil.ReadFile(certFile)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed reading CA cert: %w", err)
				}
				ctx.log.Infof("Creating new root CA cert and saving as ca.crt and ca.key in %v", certDirAbs)
				kp, err := chrome.GenerateReplacementRootCA(len(existingCA), nil, nil, ctx.log.Debugf)
				if err != nil {
					return fmt.Errorf("failed generating replacement root CA: %w", err)
				}
				if err = kp.PersistToFiles(certFile, filepath.Join(certDirAbs, "ca.key")); err != nil {
					return fmt.Errorf("failed persisting replacement root CA: %w", err)
				}
				certBytes = kp.EncodeCertPEM()
			}
			certByteBlock, _ := pem.Decode(certBytes)
			// Find lib and patch
			lib, err := chrome.FindPatchableLib(ctx.cmd.Flags().Arg(0), existingCA)
			if err != nil {
				return err
			}
			ctx.log.Infof("Patching library at: %v", lib.Path())
			return lib.Patch(certByteBlock.Bytes)
		},
	)
}
