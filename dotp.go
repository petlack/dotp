package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const TIMESTEP_DURATION_SECONDS = 30

type Config struct {
	// The name of the account for which the TOTP code is generated, e.g. foo@bar.com
	AccountName string
	// The name of the service that issued the TOTP secret, e.g. Google
	Issuer string
	// The name of the environment variable holding the secret
	SecretEnv string
	// File descriptor for the secret
	SecretFd int
	// File containing the secret
	SecretFile string
	// Read secret from stdin
	SecretStdin bool
	// Plaintext secret value
	SecretUnsafeValue string
}

type Colors struct {
	Bold, Green, Neutral, Red, Reset, ResetUnderline, Underline, Yellow string
}

var Clr = Colors{
	Bold:           "\033[1;39m",
	Green:          "\033[1;92m",
	Neutral:        "\033[0;97m",
	Red:            "\033[1;91m",
	Reset:          "\033[0;39m",
	ResetUnderline: "\033[24m",
	Underline:      "\033[4m",
	Yellow:         "\033[1;93m",
}

func LoadConfig(args []string) (*Config, []string) {
	configFlagSet := flag.NewFlagSet("config", flag.ExitOnError)

	accountName := configFlagSet.String("account", "", "Account Name")
	issuer := configFlagSet.String("issuer", "", "Issuer")
	secretUnsafeValue := configFlagSet.String("secret-unsafe-value", "", "Secret")
	secretEnv := configFlagSet.String("secret-env", "", "Name of the environment variable containing the secret")
	secretFd := configFlagSet.String("secret-fd", "", "File descriptor for the secret")
	secretFile := configFlagSet.String("secret-file", "", "File containing the secret")
	secretStdin := configFlagSet.Bool("secret-stdin", false, "Read secret from stdin")

	configFlagSet.Parse(args)

	fd := -1

	if secretFd != nil && *secretFd != "" {
		parsedFd, err := strconv.Atoi(*secretFd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid value for secretFd: %s\n", *secretFd)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fd = parsedFd
	}

	config := Config{
		AccountName:       *accountName,
		Issuer:            *issuer,
		SecretEnv:         *secretEnv,
		SecretFd:          fd,
		SecretFile:        *secretFile,
		SecretStdin:       *secretStdin,
		SecretUnsafeValue: *secretUnsafeValue,
	}

	positionalArgs := configFlagSet.Args()

	return &config, positionalArgs
}

func LoadSecret(config *Config) string {
	if config.SecretUnsafeValue != "" {
		fmt.Fprintf(os.Stderr, "Using secret passed as an argument\n")
		return config.SecretUnsafeValue
	}
	if config.SecretEnv != "" {
		if envValue := os.Getenv(config.SecretEnv); envValue != "" {
			fmt.Fprintf(os.Stderr, "Using secret from environment variable %s\n", config.SecretEnv)
			return envValue
		}
	}
	if config.SecretFile != "" {
		if secretBytes, err := os.ReadFile(config.SecretFile); err == nil {
			fmt.Fprintf(os.Stderr, "Reading secret from file %s\n", config.SecretFile)
			return string(secretBytes)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading secret file %s: %v\n", config.SecretFile, err)
			os.Exit(1)
		}
	}
	if config.SecretFd > 0 {
		fmt.Fprintf(os.Stderr, "Reading secret from file descriptor %d\n", config.SecretFd)
		secretBytes, _ := io.ReadAll(os.NewFile(uintptr(config.SecretFd), "secret"))
		return string(secretBytes)
	}
	if config.SecretStdin {
		secretBytes, _ := io.ReadAll(os.Stdin)
		fmt.Fprintf(os.Stderr, "Reading secret from stdin\n")
		return string(secretBytes)
	}
	fmt.Fprintf(os.Stderr, "%sWARNING: Using random secret.\nTo use your secret, provide one of\n\t--secret-env\n\t--secret-file\n\t--secret-unsafe-value\nRun dotp --help for more information.%s\n", Clr.Yellow, Clr.Reset)
	if secret, err := GenerateSecret(); err == nil {
		return secret
	}
	return ""
}

// GenerateSecret generates a random secret key for TOTP
func GenerateSecret() (string, error) {
	var secret [10]byte
	_, err := rand.Read(secret[:])
	if err != nil {
		return "", fmt.Errorf("Error generating secret: %v", err)
	}
	return base32.StdEncoding.EncodeToString(secret[:]), nil
}

// GenerateTotp generates a TOTP (Time-Based One-Time Password)
// code for a given secret and time. Returns a 6-digit string
func GenerateTotp(secretBase32 string, t time.Time) (string, error) {
	secretBytes, err := base32.StdEncoding.DecodeString(secretBase32)
	if err != nil {
		return "", fmt.Errorf("Error decoding secret: %v", err)
	}
	timeStep := t.Unix() / TIMESTEP_DURATION_SECONDS

	// Convert the time step to an 8-byte array in big-endian format
	var timeStepBytes [8]byte
	binary.BigEndian.PutUint64(timeStepBytes[:], uint64(timeStep))

	hmacHash := hmac.New(sha1.New, secretBytes)
	hmacHash.Write(timeStepBytes[:])
	hash := hmacHash.Sum(nil)

	lastByte := hash[len(hash)-1]
	// Use only the last 4 bits of the last byte as an offset (value between 0 and 15)
	offset := lastByte & 0b00001111
	truncatedHash := hash[offset : offset+4]

	// 0x7FFFFFFF is 01111111_11111111_11111111_11111111 in binary
	// AND sets the most significant bit to 0, making the number positive
	binaryCode := binary.BigEndian.Uint32(truncatedHash) & 0x7FFF_FFFF
	totpCode := binaryCode % 1_000_000

	// Return the TOTP code as a zero-padded string
	return fmt.Sprintf("%06d", totpCode), nil
}

// GenerateTotpUri generates a URI for provisioning a TOTP Authenticator apps.
// Example: otpauth://totp/issuer:accountName?secret=SECRET&issuer=issuer
func GenerateTotpUri(secret, accountName, issuer string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	uri := url.URL{
		Scheme:   "otpauth",
		Host:     "totp",
		Path:     fmt.Sprintf("/%s:%s", issuer, accountName),
		RawQuery: v.Encode(),
	}
	return uri.String()
}

// ValidateTotp validates a given TOTP code against a secret
func ValidateTotp(secret, code string) bool {
	totp, _ := GenerateTotp(secret, time.Now())
	return totp == code
}

func WatchTotp(secret string) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		ClearLine()
		ClearLine()
		now := time.Now()
		totp, _ := GenerateTotp(secret, now)
		PrintTotp(totp)
	}
}

func GetRemainingSeconds(t time.Time) int64 {
	return TIMESTEP_DURATION_SECONDS - t.Unix()%TIMESTEP_DURATION_SECONDS
}

func GetRemainingProgress(t time.Time) float64 {
	timestepDurationNano := TIMESTEP_DURATION_SECONDS * int64(time.Second)
	currentTimestepRemainder := t.UnixNano() % timestepDurationNano
	progress := float64(currentTimestepRemainder) / float64(timestepDurationNano)
	return progress
}

func PrintTotp(totp string) {
	now := time.Now()
	remainingSeconds := GetRemainingSeconds(now)
	progress := GetRemainingProgress(now)
	totpColor := Clr.Green
	progressColor := Clr.Reset
	if remainingSeconds <= 5 {
		totpColor = Clr.Red
		progressColor = Clr.Red
	} else if remainingSeconds <= 10 {
		totpColor = Clr.Yellow
		progressColor = Clr.Yellow
	}
	fmt.Printf(
		"%s%s%s\n%s%s%s  %s(%ds)%s\n",
		totpColor, totp, Clr.Reset,
		progressColor, ProgressBar(1-progress, 6), Clr.Reset,
		Clr.Neutral, remainingSeconds, Clr.Reset,
	)
}

func ProgressBar(progress float64, width int) string {
	blockIndex := int(progress * float64(width))
	// blockProgressChars := []string{"\u2588", "\u2589", "\u258a", "\u258b", "\u258c", "\u258d", "\u258e", "\u258f"}
	blockProgressChars := []string{"█", "▉", "▊", "▋", "▌", "▍", "▎", "▏"}
	numBlocks := width * len(blockProgressChars)
	blockProgress := int((1-progress)*float64(numBlocks)) % len(blockProgressChars)
	bar := strings.Builder{}
	for i := range width {
		if i < blockIndex {
			bar.WriteRune('█')
		} else if i == blockIndex {
			bar.WriteString(blockProgressChars[blockProgress])
		} else {
			bar.WriteRune('░')
		}
	}
	return bar.String()
}

func ClearLine() {
	// \033[1A - Move the cursor up by 1 line
	// \033[2K - Clear the entire line
	fmt.Print("\033[1A\033[2K")
}
