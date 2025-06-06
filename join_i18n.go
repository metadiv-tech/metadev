package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var joinI18nCmd = &cobra.Command{
	Use:   "join_i18n [files...]",
	Short: "Join multiple i18n JSON files into one",
	Long:  "Merge multiple i18n JSON files into a single file, removing duplicates and prioritizing non-empty values",
	Args:  cobra.MinimumNArgs(1),
	Run:   runJoinI18n,
}

var outputFile string

func init() {
	joinI18nCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (defaults to random name if not specified)")
}

func runJoinI18n(cmd *cobra.Command, args []string) {
	// Validate input files
	for _, file := range args {
		if !strings.HasSuffix(file, ".json") {
			fmt.Fprintf(os.Stderr, "Error: %s is not a JSON file\n", file)
			os.Exit(1)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: File %s does not exist\n", file)
			os.Exit(1)
		}
	}

	// Merge JSON files
	merged, err := mergeJSONFiles(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging files: %v\n", err)
		os.Exit(1)
	}

	// Determine output file name
	output := outputFile
	if output == "" {
		output = generateRandomFileName()
	}
	if !strings.HasSuffix(output, ".json") {
		output += ".json"
	}

	// Write merged content to output file
	err = writeJSONFile(output, merged)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged %d files into %s with %d keys\n", len(args), output, len(merged))
}

func mergeJSONFiles(files []string) (map[string]string, error) {
	merged := make(map[string]string)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", file, err)
		}

		var data map[string]string
		err = json.Unmarshal(content, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON in file %s: %v", file, err)
		}

		// Merge data, prioritizing non-empty values
		for key, value := range data {
			existingValue, exists := merged[key]
			if !exists {
				// Key doesn't exist, add it
				merged[key] = value
			} else {
				// Key exists, keep non-empty value
				if existingValue == "" && value != "" {
					merged[key] = value
				}
				// If existing value is not empty, keep it (skip empty values)
			}
		}
	}

	return merged, nil
}

func writeJSONFile(filename string, data map[string]string) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create ordered map for JSON marshaling
	orderedData := make(map[string]string)
	for _, key := range keys {
		orderedData[key] = data[key]
	}

	jsonData, err := json.MarshalIndent(orderedData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

func generateRandomFileName() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "merged_i18n_" + hex.EncodeToString(bytes)
}
