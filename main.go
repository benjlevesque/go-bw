package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/benjlevesque/go-bw/internal/bitwarden"
	"github.com/benjlevesque/go-bw/internal/login"
	"github.com/manifoldco/promptui"
)

var (
	versionFlag = false
	version     = "dev"
	commit      = "none"
)

func main() {
	flag.BoolVar(&versionFlag, "version", false, "Display version")
	flag.Parse()

	if versionFlag {
		fmt.Printf("Version %s, commit %s\n", version, commit)
		return
	}

	if flag.NArg() < 1 {
		log.Fatal("Missing required argument: path to json file")
	}

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	entries, err := bitwarden.ParseDatabaseJson(file)
	if err != nil {
		log.Fatalf("Failed to parse database: %v", err)
	}

	selected, err := runLoginSelectionWizard(entries)
	if err != nil {
		log.Fatalf("Failed to select login entry: %v", err)
	}
	runCopyToClipboardWizard(selected)

}

func runLoginSelectionWizard(entries []login.LoginEntry) (login.LoginEntry, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ .Fullname }}",
		Active:   "> {{ .Fullname | cyan }}",
		Inactive: "  {{ .Fullname | cyan }}",
		Details: `
--------- Login ----------
{{ "Name:" | faint }}	{{ .Name }}
{{if .Login}}{{ "Login:" | faint }}	{{ .Login }}
{{end}}{{if .URIs}}{{ "URIs:" | faint }}	{{ range .URIs }}{{ . }} {{ end }}
{{end}}{{if .TOTPSecret}}{{ "TOTP:" | faint }}	{{ .TOTP }}
{{end}}{{if .Password}}{{ "Password:" | faint }}	{{ .Password }}{{end}}`}

	templates.Selected = templates.Details

	prompt := promptui.Select{
		Label:             "Select item",
		Items:             entries,
		StartInSearchMode: true,
		Templates:         templates,
		Keys: &promptui.SelectKeys{
			Search:   promptui.Key{Code: '?', Display: "?"},
			Next:     promptui.Key{Code: promptui.KeyNext, Display: promptui.KeyNextDisplay},
			Prev:     promptui.Key{Code: promptui.KeyPrev, Display: promptui.KeyPrevDisplay},
			PageUp:   promptui.Key{Code: promptui.KeyForward, Display: promptui.KeyForwardDisplay},
			PageDown: promptui.Key{Code: promptui.KeyForward, Display: promptui.KeyForwardDisplay},
		},
		Searcher: func(input string, index int) bool {
			entry := entries[index]
			value := entry.Fullname()
			input = strings.ToLower(input)
			return strings.Contains(strings.ToLower(value), input)
		},
	}

	index, _, err := prompt.Run()

	if err != nil {
		return login.LoginEntry{}, err
	}
	return entries[index], nil
}

func runCopyToClipboardWizard(selected login.LoginEntry) {
	if selected.Login == "" && selected.Password == "" && selected.TOTPSecret == "" {
		return
	}

	options := []string{"all"}
	if selected.Login != "" {
		options = append(options, "login")

	}
	if selected.Password != "" {
		options = append(options, "password")
	}
	if selected.TOTPSecret != "" {
		options = append(options, "totp")
	}
	if len(options) == 1 {
		return
	}

	prompt := promptui.Select{
		Label:        "Copy to clipboard?",
		Items:        options,
		HideSelected: true,
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Printf("Prompt failed: %v\n", err)
		return
	}
	switch strings.ToLower(result) {
	case "login":
		copyToClipboard(selected.Login, "Login copied to clipboard")
	case "password":
		copyToClipboard(selected.Password, "Password copied to clipboard")
	case "totp":
		copyToClipboard(selected.TOTP(), "TOTP copied to clipboard")
	case "all":
		reader := bufio.NewReader(os.Stdin)
		if selected.Login != "" {
			copyToClipboard(selected.Login, "Login copied to clipboard. Press enter to continue...")
			reader.ReadLine()
		}
		if selected.Password != "" {
			if selected.TOTPSecret != "" {
				copyToClipboard(selected.Password, "Password copied to clipboard. Press enter to continue...")
				reader.ReadLine()
				copyToClipboard(selected.TOTP(), "TOTP copied to clipboard")
			} else {
				copyToClipboard(selected.Password, "Password copied to clipboard.")
			}
		}

	}

}

func copyToClipboard(text string, successText string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}
	fmt.Println(successText)
}
