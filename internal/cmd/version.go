package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = "SNAPSHOT"

func versionCmd() *cobra.Command {

	cmd := cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	return &cmd

}
