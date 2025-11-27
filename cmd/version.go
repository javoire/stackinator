package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (set by goreleaser at build time)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, and build date of this stack binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stack version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
