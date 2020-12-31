package cmd

import (
	"github.com/cretz/takecast/pkg/chrome"
	"github.com/spf13/cobra"
)

func unpatchCmd() *cobra.Command {
	return applyRun(
		&cobra.Command{
			Use:   "unpatch [path to chrome parent dir]",
			Short: "Unpatch an already-patched Chrome",
			Args:  cobra.ExactArgs(1),
		},
		func(ctx *rootContext) error {
			// Find lib and unpatch
			lib, err := chrome.FindUnpatchableLib(ctx.cmd.Flags().Arg(0), chrome.LoadExistingRootCADERBytes())
			if err != nil {
				return err
			}
			ctx.log.Infof("Unpatching library from %v to %v", lib.Path(), lib.OrigPath())
			return lib.Unpatch()
		},
	)
}
