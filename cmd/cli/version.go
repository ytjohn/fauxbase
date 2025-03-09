package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ytjohn/fauxbase/pkg/about"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", about.AppName, about.Version)
		fmt.Printf("%s\n", about.Copyright)
		fmt.Printf("%s\n", about.URL)
	},
}
