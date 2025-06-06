package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "metadev",
	Short: "A CLI tool to help developers",
	Long:  "metadev is a CLI tool designed to help developers with various development tasks",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(i18nCmd)
	rootCmd.AddCommand(joinI18nCmd)
}
