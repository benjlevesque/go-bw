package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/manifoldco/promptui"
	"github.com/pquerna/otp/totp"
)

type LoginEntry struct {
	ID         string
	Folder     string
	Name       string
	Login      string
	Password   string
	URIs       []string
	TOTPSecret string
}

func (e LoginEntry) Fullname() string {
	// if e.Folder != "" {
	// 	return fmt.Sprintf("%s/%s", e.Folder, e.Name)
	// }
	return e.Name
}

func (e LoginEntry) TOTP() string {
	if e.TOTPSecret == "" {
		return ""
	}
	code, err := totp.GenerateCode(e.TOTPSecret, time.Now())
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
	return code
}

type BitwardenDB struct {
	Folders []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"folders"`
	Items []struct {
		ID       string `json:"id"`
		FolderId string `json:"folderId"`
		Name     string `json:"name"`
		Login    struct {
			Username string `json:"username"`
			Password string `json:"password"`
			URIs     []struct {
				URI string `json:"uri"`
			} `json:"urIs"`
			TOTP string `json:"totp"`
		} `json:"login"`
	} `json:"items"`
}

func readDatabaseJson(path string) []byte {

	var buf []byte
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		buf = append(buf, scanner.Bytes()...)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return buf
}

var (
	copyTOTP     = false
	copyPassword = false
	copyLogin    = false
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionFlag = flag.Bool("version", false, "Display version")

func main() {
	if *versionFlag {
		fmt.Printf("Version %s, commit %s, built at %s\n", version, commit, date)
		return
	}

	flag.BoolVar(&copyTOTP, "totp", false, "Copy TOTP to clipboard")
	flag.BoolVar(&copyPassword, "password", false, "Copy password to clipboard")
	flag.BoolVar(&copyLogin, "login", false, "Copy login to clipboard")
	flag.Parse()

	if (copyTOTP && copyPassword) || (copyTOTP && copyLogin) || (copyPassword && copyLogin) {
		log.Fatal("Only one of --totp, --password, or --login can be specified at a time")
	}

	if flag.NArg() < 1 {
		log.Fatal("Missing required argument: path to json file")
	}
	jsonData := readDatabaseJson(flag.Arg(0))
	db := BitwardenDB{}
	err := json.Unmarshal(jsonData, &db)
	if err != nil {
		log.Fatalf("Unable to unmarshal data: %v", err)
	}
	folders := make(map[string]string)
	for _, folder := range db.Folders {
		folders[folder.ID] = folder.Name
	}

	entries := []LoginEntry{}
	for _, entry := range db.Items {
		uris := []string{}
		for _, uri := range entry.Login.URIs {
			uris = append(uris, uri.URI)
		}
		entries = append(entries, LoginEntry{
			ID:         entry.ID,
			Name:       entry.Name,
			Login:      entry.Login.Username,
			Password:   entry.Login.Password,
			URIs:       uris,
			Folder:     folders[entry.FolderId],
			TOTPSecret: entry.Login.TOTP,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Fullname() < entries[j].Fullname()
	})

	templates := &promptui.SelectTemplates{
		Label:    "{{ .Fullname }}",
		Active:   "> {{ .Fullname | cyan }}",
		Inactive: "  {{ .Fullname | cyan }}",
		Details: `
--------- Login ----------
{{ "Name:" | faint }}	{{ .Name }}
{{if .Login}}{{ "Login:" | faint }}	{{ .Login }}
{{end}}{{if .URIs}}{{ "URIs:" | faint }}	{{ range .URIs }}{{ . }} {{ end }}
{{end}}{{if .TOTPSecret}}{{ "TOTP:" | faint }}	{{ .TOTPSecret }}
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
		// HideSelected:      true,
		Searcher: func(input string, index int) bool {
			entry := entries[index]
			name := entry.Name
			folder := entry.Folder
			input = strings.ToLower(input)
			return strings.Contains(strings.ToLower(name), input) || strings.Contains(strings.ToLower(folder), input)
		},
	}

	index, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	selected := entries[index]

	if selected.Login == "" && selected.Password == "" && selected.TOTPSecret == "" {
		return
	}
	if !copyLogin && !copyPassword && !copyTOTP {
		options := []string{}
		if selected.Login != "" {
			options = append(options, "login")
		}
		if selected.Password != "" {
			options = append(options, "password")
		}
		if selected.TOTPSecret != "" {
			options = append(options, "totp")
		}
		if len(options) == 0 {
			return
		}

		prompt := promptui.Select{
			Label: "Copy to clipboard?",
			Items: options,
		}
		_, result, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed! %v\n", err)
			return
		}
		switch strings.ToLower(result) {
		case "login":
			copyLogin = true
		case "password":
			copyPassword = true
		case "totp":
			copyTOTP = true
		}
	}

	textToCopy := ""
	if copyLogin {
		textToCopy = selected.Login
	}
	if copyPassword {
		textToCopy = selected.Password
	}
	if copyTOTP {
		textToCopy = selected.TOTP()
	}

	if textToCopy != "" {
		err := clipboard.WriteAll(textToCopy)
		if err != nil {
			log.Fatalf("Failed to copy to clipboard: %v", err)
		}
	}
}
