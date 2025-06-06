package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type TranslationKey struct {
	Key       string
	Namespace string
	File      string
}

var i18nCmd = &cobra.Command{
	Use:   "i18n",
	Short: "Extract translation keys from React TSX files",
	Long:  "Parse all .tsx files in the project and extract useTranslation keys to generate JSON translation files",
	Run:   runI18n,
}

func runI18n(cmd *cobra.Command, args []string) {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	keys, err := extractTranslationKeys(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting translation keys: %v\n", err)
		os.Exit(1)
	}

	err = setupI18nDirectory(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up .i18n directory: %v\n", err)
		os.Exit(1)
	}

	err = updateGitignore(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating .gitignore: %v\n", err)
		os.Exit(1)
	}

	err = generateTranslationFiles(workDir, keys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating translation files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully extracted %d translation keys and generated translation files\n", len(keys))
}

func extractTranslationKeys(rootDir string) ([]TranslationKey, error) {
	// First pass: build global mapping of t-variables to namespaces
	globalTranslationMap := make(map[string]string)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}

		if !strings.HasSuffix(path, ".tsx") {
			return nil
		}

		fileMap, err := extractUseTranslationDeclarations(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error parsing declarations in file %s: %v\n", path, err)
			return nil
		}

		// Merge into global map
		for tVar, namespace := range fileMap {
			globalTranslationMap[tVar] = namespace
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: extract translation keys using the global mapping
	var keys []TranslationKey

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}

		if !strings.HasSuffix(path, ".tsx") {
			return nil
		}

		fileKeys, err := parseFileWithMapping(path, globalTranslationMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error parsing file %s: %v\n", path, err)
			return nil
		}

		keys = append(keys, fileKeys...)
		return nil
	})

	return keys, err
}

func shouldSkipDir(dirName string) bool {
	skipDirs := []string{"node_modules", "vendor", ".git", ".next", "dist", "build"}
	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}
	return false
}

func parseFile(filePath string) ([]TranslationKey, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var keys []TranslationKey

	useTranslationRegex := regexp.MustCompile(`const\s*{\s*t:\s*(\w+)\s*}\s*=\s*useTranslation\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	matches := useTranslationRegex.FindAllStringSubmatch(string(content), -1)

	translationCallMap := make(map[string]string)
	for _, match := range matches {
		if len(match) >= 3 {
			tVariable := match[1]
			namespace := match[2]
			translationCallMap[tVariable] = namespace
		}
	}

	for tVar, namespace := range translationCallMap {
		callRegex := regexp.MustCompile(fmt.Sprintf(`%s\s*\(\s*['"]([^'"]+)['"]\s*\)`, regexp.QuoteMeta(tVar)))
		callMatches := callRegex.FindAllStringSubmatch(string(content), -1)

		for _, callMatch := range callMatches {
			if len(callMatch) >= 2 {
				key := callMatch[1]
				keys = append(keys, TranslationKey{
					Key:       key,
					Namespace: namespace,
					File:      filePath,
				})
			}
		}
	}

	return keys, nil
}

func extractUseTranslationDeclarations(filePath string) (map[string]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	translationMap := make(map[string]string)
	contentStr := string(content)

	// Find all useTranslation declarations: const { t: tVariable } = useTranslation('namespace')
	useTranslationRegex := regexp.MustCompile(`const\s*{\s*t:\s*(\w+)\s*}\s*=\s*useTranslation\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	matches := useTranslationRegex.FindAllStringSubmatch(contentStr, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tVariable := match[1]
			namespace := match[2]
			translationMap[tVariable] = namespace
		}
	}

	return translationMap, nil
}

func parseFileWithMapping(filePath string, globalMapping map[string]string) ([]TranslationKey, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var keys []TranslationKey
	contentStr := string(content)

	// Find all translation function calls (pattern: tSomething('key')) - must start with 't'
	allCallsRegex := regexp.MustCompile(`\b(t[A-Za-z][A-Za-z0-9]*)\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	allCallMatches := allCallsRegex.FindAllStringSubmatch(contentStr, -1)

	for _, callMatch := range allCallMatches {
		if len(callMatch) >= 3 {
			tVariable := callMatch[1]
			key := callMatch[2]

			var namespace string
			if ns, exists := globalMapping[tVariable]; exists {
				// Use the namespace from global mapping
				namespace = ns
			} else {
				// Infer namespace from variable name (e.g., tCommon -> common)
				if strings.HasPrefix(tVariable, "t") && len(tVariable) > 1 {
					namespace = strings.ToLower(tVariable[1:])
				} else {
					namespace = "common" // fallback to common
				}
			}

			keys = append(keys, TranslationKey{
				Key:       key,
				Namespace: namespace,
				File:      filePath,
			})
		}
	}

	return keys, nil
}

func generateTranslationFiles(rootDir string, keys []TranslationKey) error {
	namespaceKeys := make(map[string][]string)

	for _, key := range keys {
		namespaceKeys[key.Namespace] = append(namespaceKeys[key.Namespace], key.Key)
	}

	for namespace, keysList := range namespaceKeys {
		translationMap := make(map[string]string)

		uniqueKeys := make(map[string]bool)
		for _, key := range keysList {
			uniqueKeys[key] = true
		}

		for key := range uniqueKeys {
			translationMap[key] = ""
		}

		jsonData, err := json.MarshalIndent(translationMap, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON for namespace %s: %v", namespace, err)
		}

		i18nDir := filepath.Join(rootDir, ".i18n")
		fileName := filepath.Join(i18nDir, fmt.Sprintf("%s.json", namespace))
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("error writing file %s: %v", fileName, err)
		}

		fmt.Printf("Generated %s with %d keys\n", fileName, len(translationMap))
	}

	return nil
}

func setupI18nDirectory(rootDir string) error {
	i18nDir := filepath.Join(rootDir, ".i18n")
	err := os.MkdirAll(i18nDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating .i18n directory: %v", err)
	}
	return nil
}

func updateGitignore(rootDir string) error {
	gitignorePath := filepath.Join(rootDir, ".gitignore")

	gitignoreExists := true
	content, err := os.ReadFile(gitignorePath)
	if os.IsNotExist(err) {
		gitignoreExists = false
		content = []byte{}
	} else if err != nil {
		return fmt.Errorf("error reading .gitignore: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, ".i18n/") {
		return nil
	}

	if !gitignoreExists {
		contentStr = ".i18n/\n"
	} else {
		if !strings.HasSuffix(contentStr, "\n") {
			contentStr += "\n"
		}
		contentStr += ".i18n/\n"
	}

	err = os.WriteFile(gitignorePath, []byte(contentStr), 0644)
	if err != nil {
		return fmt.Errorf("error writing .gitignore: %v", err)
	}

	if !gitignoreExists {
		fmt.Println("Created .gitignore and added .i18n/")
	} else {
		fmt.Println("Added .i18n/ to .gitignore")
	}

	return nil
}
