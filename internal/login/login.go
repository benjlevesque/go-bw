package login

import (
	"log"
	"time"

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
