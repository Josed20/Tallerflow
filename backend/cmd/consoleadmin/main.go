package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformconsole "github.com/Josed20/Tallerflow/backend/internal/console"
	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "create" {
		log.Fatal("usage: tallerflow-consoleadmin create --email admin@example.com --password-stdin")
	}
	var email string
	for index := 2; index < len(os.Args)-1; index++ {
		if os.Args[index] == "--email" {
			email = strings.ToLower(strings.TrimSpace(os.Args[index+1]))
		}
	}
	if email == "" {
		log.Fatal("--email is required")
	}
	databaseURL := os.Getenv("TF_CONSOLE_DATABASE_URL")
	encryptionKey := os.Getenv("TF_CONSOLE_TOTP_KEY")
	if databaseURL == "" || encryptionKey == "" {
		log.Fatal("TF_CONSOLE_DATABASE_URL and TF_CONSOLE_TOTP_KEY are required")
	}
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(password) == 0 {
		log.Fatal("read password: ", err)
	}
	password = strings.TrimSpace(password)
	if len(password) < 12 {
		log.Fatal("password must contain at least 12 characters")
	}
	hash, err := auth.NewPasswordHasher(auth.DefaultPasswordParams()).Hash(password)
	if err != nil {
		log.Fatal(err)
	}
	secret, err := platformconsole.GenerateSecret()
	if err != nil {
		log.Fatal(err)
	}
	encrypted, err := platformconsole.EncryptSecret(encryptionKey, secret)
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close(db)
	if err := db.Exec("INSERT INTO console_admins (id, email, password_hash, totp_secret_ciphertext) VALUES (?, ?, ?, ?)", uuid.New(), email, hash, encrypted).Error; err != nil {
		log.Fatal(err)
	}
	otpauth := "otpauth://totp/TallerFlow:" + url.QueryEscape(email) + "?secret=" + secret + "&issuer=TallerFlow&digits=6&period=30"
	fmt.Println("Administrador creado. Agrega esta URI una sola vez en tu aplicación autenticadora:")
	fmt.Println(otpauth)
}
