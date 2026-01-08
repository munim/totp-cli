package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"

	"bufio"
	"encoding/base32"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/spf13/cobra"
	"github.com/xlzd/gotp"
	"github.com/zalando/go-keyring"
)

const serviceName = "totp"

type indexFile struct {
	Names []string `json:"names"`
}

func indexFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".totp.json"), nil
}

func readIndex() (indexFile, error) {
	path, err := indexFilePath()
	if err != nil {
		return indexFile{}, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return indexFile{}, nil
		}
		return indexFile{}, err
	}

	var idx indexFile
	if err := json.Unmarshal(b, &idx); err != nil {
		return indexFile{}, err
	}
	return idx, nil
}

func writeIndex(idx indexFile) error {
	path, err := indexFilePath()
	if err != nil {
		return err
	}

	sort.Strings(idx.Names)
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

func addNameToIndex(name string) error {
	idx, err := readIndex()
	if err != nil {
		return err
	}

	for _, n := range idx.Names {
		if n == name {
			return nil
		}
	}
	idx.Names = append(idx.Names, name)
	return writeIndex(idx)
}

func removeNameFromIndex(name string) error {
	idx, err := readIndex()
	if err != nil {
		return err
	}

	out := idx.Names[:0]
	for _, n := range idx.Names {
		if n != name {
			out = append(out, n)
		}
	}
	idx.Names = out
	return writeIndex(idx)
}

func normalizeAndValidateSecret(secret string) (string, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if normalized == "" {
		return "", errors.New("No secret was given")
	}
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized); err != nil {
		return "", errors.New("Invalid secret (expected Base32)")
	}
	return normalized, nil
}

func addItem(name, secret string) error {
	if err := keyring.Set(serviceName, name, secret); err != nil {
		if errors.Is(err, keyring.ErrSetDataTooBig) {
			return fmt.Errorf("secret too large to store in system keyring: %w", err)
		}
		return err
	}
	return addNameToIndex(name)
}

func getItem(name string) (string, error) {
	secret, err := keyring.Get(serviceName, name)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", errors.New("Given name is not found")
		}
		return "", err
	}
	return secret, nil
}

func deleteItem(name string) error {
	err := keyring.Delete(serviceName, name)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return removeNameFromIndex(name)
}

func listItems() ([]string, error) {
	idx, err := readIndex()
	if err != nil {
		return nil, err
	}

	var kept []string
	for _, name := range idx.Names {
		_, err := keyring.Get(serviceName, name)
		if err == nil {
			kept = append(kept, name)
			continue
		}
		if errors.Is(err, keyring.ErrNotFound) {
			continue
		}
		return nil, err
	}
	idx.Names = kept
	if err := writeIndex(idx); err != nil {
		return nil, err
	}

	return idx.Names, nil
}

func nameExists(name string) (bool, error) {
	_, err := keyring.Get(serviceName, name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func promptNewName(initial string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	name := initial
	for {
		exists, err := nameExists(name)
		if err != nil {
			return "", err
		}
		if !exists {
			return name, nil
		}

		fmt.Printf("Name \"%v\" already exists. Type new name: ", name)
		line, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		name = strings.TrimSpace(line)
		if name == "" {
			continue
		}
	}
}

func main() {
	var useBarcodeHintWhenScan bool

	var cmdScan = &cobra.Command{
		Use:   "scan <name> <image>",
		Short: "Scan a QR code image",
		Long:  `Scan a QR code image and store it to the system keyring.`,
		Args:  cobra.ExactArgs(2),

		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			path := args[1]

			// open and decode image file
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			img, _, err := image.Decode(file)
			if err != nil {
				return err
			}

			// prepare BinaryBitmap
			bmp, err := gozxing.NewBinaryBitmapFromImage(img)
			if err != nil {
				return err
			}

			// decode image
			qrReader := qrcode.NewQRCodeReader()

			var hint map[gozxing.DecodeHintType]interface{}
			if useBarcodeHintWhenScan {
				hint = map[gozxing.DecodeHintType]interface{}{
					gozxing.DecodeHintType_PURE_BARCODE: struct{}{},
				}
			}

			result, err := qrReader.Decode(bmp, hint)
			if err != nil {
				return err
			}

			// parse TOTP URL
			parsed, err := url.Parse(result.GetText())
			if err != nil {
				return err
			}
			secret, err := normalizeAndValidateSecret(parsed.Query().Get("secret"))
			if err != nil {
				return err
			}
			if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
				return errors.New("Given QR code is not for TOTP")
			}

			name, err = promptNewName(name)
			if err != nil {
				return err
			}

			err = addItem(name, secret)
			if err != nil {
				return err
			}
			fmt.Printf("Given QR code successfully registered as \"%v\".\n", name)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 1 {
				return nil, cobra.ShellCompDirectiveDefault
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmdScan.Flags().BoolVarP(
		&useBarcodeHintWhenScan,
		"barcode",
		"b",
		false,
		"use PURE_BARCODE hint for decoding. this flag maybe solves FormatException",
	)

	var cmdAdd = &cobra.Command{
		Use:   "add <name>",
		Short: "Manually add a secret to the system keyring",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := promptNewName(args[0])
			if err != nil {
				return err
			}

			// Read secret from stdin
			var secret string
			fmt.Print("Type secret: ")
			fmt.Scanln(&secret)

			secret, err = normalizeAndValidateSecret(secret)
			if err != nil {
				return err
			}
			fmt.Printf("Current code: %v\n", gotp.NewDefaultTOTP(secret).Now())

			err = addItem(name, secret)
			if err != nil {
				return err
			}
			fmt.Printf("Given secret successfully registered as \"%v\".\n", name)
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	var cmdList = &cobra.Command{
		Use:   "list",
		Short: "List all registered TOTP codes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := listItems()
			if err != nil {
				return err
			}

			for _, name := range names {
				fmt.Println(name)
			}
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	var cmdGet = &cobra.Command{
		Use:   "get <name>",
		Short: "Get a TOTP code",
		Long:  "Get a TOTP code from the system keyring.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			secret, err := getItem(name)
			if err != nil {
				return err
			}

			fmt.Println(gotp.NewDefaultTOTP(secret).Now())
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			names, err := listItems()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			return names, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var cmdDelete = &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a TOTP code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			err := deleteItem(name)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully deleted \"%v\".\n", name)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			names, err := listItems()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			return names, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var cmdTemp = &cobra.Command{
		Use:   "temp",
		Short: "Get a TOTP code from a secret without saving it to the keyring",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var secret string
			fmt.Print("Type secret: ")
			fmt.Scanln(&secret)

			secret, err := normalizeAndValidateSecret(secret)
			if err != nil {
				return err
			}

			fmt.Println(gotp.NewDefaultTOTP(secret).Now())
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	var rootCmd = &cobra.Command{Use: "totp", Short: "Simple TOTP CLI, powered by the system keyring", Version: "1.1.3"}
	rootCmd.AddCommand(cmdScan, cmdAdd, cmdList, cmdGet, cmdDelete, cmdTemp)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
