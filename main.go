package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"
)

//go:embed version.txt
var versionFile embed.FS
var Version string

func init() {
	versionData, err := fs.ReadFile(versionFile, "version.txt")
	if err != nil {
		log.Fatal(err)
	}
	Version = strings.TrimSpace(string(versionData))
}

func main() {
	args := os.Args[1:]

	action := ""
	if len(args) < 1 {
		action = "watch"
	} else {
		action = os.Args[1]
	}

	if len(args) > 0 {
		args = args[1:]
	}
	config, args := LoadConfig(args)

	switch action {

	case "help", "--help", "-h":
		fmt.Fprintf(os.Stderr, "%sdotp %sv%s%s\n", Clr.Bold, Clr.Neutral, Version, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  Simple TOTP (Time-based One-time Password) utility\n\n")
		fmt.Fprintf(os.Stderr, "%sUSAGE:%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  dotp action [options]\n\n")
		fmt.Fprintf(os.Stderr, "%sACTIONS%s:\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  %sget [%sOPTIONS%s]%s\n", Clr.Bold, Clr.Underline, Clr.ResetUnderline, Clr.Reset)
		fmt.Fprintf(os.Stderr, "    Print the current TOTP code\n")
		fmt.Fprintf(os.Stderr, "  %snew%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "    Generate a new secret key for TOTP and print it to standard output\n")
		fmt.Fprintf(os.Stderr, "  %suri [%sOPTIONS%s] --account [account] --issuer [issuer]%s\n", Clr.Bold, Clr.Underline, Clr.ResetUnderline, Clr.Reset)
		fmt.Fprintf(os.Stderr, "    Generate a URI for provisioning a TOTP Authenticator app\n")
		fmt.Fprintf(os.Stderr, "  %svalidate <CODE> [%sOPTIONS%s]%s\n", Clr.Bold, Clr.Underline, Clr.ResetUnderline, Clr.Reset)
		fmt.Fprintf(os.Stderr, "    Validate a TOTP code\n")
		fmt.Fprintf(os.Stderr, "  %swatch [%sOPTIONS%s]%s\n", Clr.Bold, Clr.Underline, Clr.ResetUnderline, Clr.Reset)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s%sOPTIONS%s:\n", Clr.Underline, Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  %s--secret-env <ENV_NAME>%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "      Name of the environment variable holding the secret\n")
		fmt.Fprintf(os.Stderr, "  %s--secret-file <PATH>%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "      Path to the file containing the secret\n")
		fmt.Fprintf(os.Stderr, "  %s--secret-stdin%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "      Read secret from standard input\n")
		fmt.Fprintf(os.Stderr, "  %s--secret-unsafe-value <SECRET>%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "      Use the secret provided as an argument\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%sEXAMPLES%s:\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "%s- Using a file%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  dotp new > ./mysecret\n")
		fmt.Fprintf(os.Stderr, "  dotp uri --account foo@bar --issuer myapp --secret-file ./mysecret\n")
		fmt.Fprintf(os.Stderr, "  dotp validate 112233 --secret-file ./mysecret\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s- Using an environment variable%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  export TOTP_SECRET=mysecret\n")
		fmt.Fprintf(os.Stderr, "  dotp uri --account foo@bar --issuer myapp --secret-env TOTP_SECRET\n")
		fmt.Fprintf(os.Stderr, "  dotp validate 112233 --secret-env TOTP_SECRET\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s- Using standard input%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  echo mysecret | dotp uri --account foo@bar --issuer myapp --secret-stdin\n")
		fmt.Fprintf(os.Stderr, "  echo mysecret | dotp validate 112233 --secret-stdin\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s- Integration with pass utility%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  dotp new | pass insert -e 'TOTP/mykey'\n")
		fmt.Fprintf(os.Stderr, "  pass show 'TOTP/mykey' | dotp get --secret-stdin\n")
		fmt.Fprintf(os.Stderr, "  pass show 'TOTP/mykey' | dotp validate --secret-stdin 112233\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s- Integration with qrencode%s\n", Clr.Bold, Clr.Reset)
		fmt.Fprintf(os.Stderr, "  pass show 'TOTP/mykey' | dotp uri --account foo@bar --issuer myapp --secret-stdin | qrencode -t ANSI\n")
		fmt.Fprintln(os.Stderr)
		os.Exit(0)

	case "get":
		secret := LoadSecret(config)
		totp, err := GenerateTotp(secret, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating TOTP: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(totp)
		os.Exit(0)

	case "new":
		secret, _ := GenerateSecret()
		fmt.Print(secret)
		os.Exit(0)

	case "uri":
		secret := LoadSecret(config)
		totpUri := GenerateTotpUri(secret, "demo-account", "demo-app")
		fmt.Print(totpUri)
		os.Exit(0)

	case "validate":
		secret := LoadSecret(config)
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: dotp validate [secret] [code]\nGiven args: %v\n", args)
			os.Exit(1)
		}
		code := args[0]
		if ValidateTotp(secret, code) {
			fmt.Fprintf(os.Stderr, "Valid code (expires in %d seconds)", GetRemainingSeconds(time.Now()))
			os.Exit(0)
		} else {
			fmt.Fprintf(os.Stderr, "Invalid code")
			os.Exit(1)
		}

	case "version", "--version", "-v":
		fmt.Print(Version)
		os.Exit(0)

	case "watch":
		secret := LoadSecret(config)
		fmt.Fprintf(os.Stderr, "%sPress Ctrl+C to exit%s\n", Clr.Neutral, Clr.Reset)
		fmt.Fprintf(os.Stderr, "Your TOTP code is:\n\n\n\n")
		WatchTotp(secret)
	}
}
