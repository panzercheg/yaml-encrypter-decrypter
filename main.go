package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

//global variables
var envWhiteSpaces int
var valuesForFormatting = false
var tmpYamlText []string

func main() {
	flagFile := flag.String(
		"filename",
		"values.yaml",
		"filename for encode/decode",
	)
	flagKey := flag.String(
		"key",
		"}tf&Wr+Nt}A9g{s",
		"AES key for encrypt/decrypt",
	)
	flagEnv := flag.String("env", "env:", "block-name for encode/decode")
	flagDebug := flag.String("debug", "false", "debug mode, print encode/decode to stdout")
	flagEncryptValue := flag.String("encrypt", "", "value to encrypt")
	flagDecryptValue := flag.String("decrypt", "", "value to decrypt")
	flagVerbose := flag.String("verbose", "false", "verbose file")
	flag.Parse()

	filename := *flagFile
	key := *flagKey
	env := *flagEnv
	debug := *flagDebug
	encryptValue := *flagEncryptValue
	decryptValue := *flagDecryptValue
	verbose := *flagVerbose
	// for @kpogonea
	const AES = "AES256:"

	// for @jaxel87, encrypt/decrypt value by flag without encrypt/decrypt file
	if encryptValue != "" {
		encrypted, err := encryptAES(key, encryptValue)
		fmt.Println(encrypted)
		if err != nil {
			log.Fatalf("something went wrong during encrypt")
		}
		os.Exit(0)
	}
	if decryptValue != "" {
		decrypted, err := decryptAES(key, decryptValue)
		fmt.Println(decrypted)
		if err != nil {
			log.Fatalf("something went wrong during decrypt")
		}
		os.Exit(0)
	}

	text := readFile(filename)
	for _, eachLn := range text {
		//show current whitespaces before character
		currentWhiteSpaces := countLeadingSpaces(eachLn)
		if envWhiteSpaces > currentWhiteSpaces {
			valuesForFormatting = false
		}
		if valuesForFormatting {
			stringArray := strings.Fields(eachLn)
			var value string
			// check if value not set
			if len(stringArray) == 1 {
				value = ""
			} else {
				value = stringArray[1]
			}
			// print whitespaces
			whitespaces := strings.Repeat(" ", currentWhiteSpaces)

			encrypted, err := encryptAES(key, value)
			if err != nil {
				log.Fatalf("something went wrong - %s", err)
			}
			matchedAesEncrypted, _ := regexp.MatchString(AES, value)
			// check file is encrypted
			if !matchedAesEncrypted {
				if verbose == "true" {
					if stringArray[0] == "#" || stringArray[0] == "# " {
						fmt.Println(eachLn)
					} else {
						if value != "" {
							fmt.Println(whitespaces + stringArray[0] + " " + AES + encrypted)
						} else {
							fmt.Println(whitespaces + stringArray[0] + value)
						}
					}
				}
				//check if line is empty
				if eachLn == "" {
					tmpYamlText = append(tmpYamlText, eachLn)
				}
				//check if line in file is comment
				if strings.HasPrefix(stringArray[0], "#") {
					tmpYamlText = append(tmpYamlText, eachLn)
				} else {
					if value != "" {
						stringArray[1] = AES + encrypted
						tmpYamlText = append(tmpYamlText, whitespaces+strings.Join(stringArray, " "))
						fmt.Println(encrypted)
					} else {
						tmpYamlText = append(tmpYamlText, whitespaces+stringArray[0]+value)
					}
				}

			} else {
				aesBeforeDecrypt := strings.ReplaceAll(value, AES, "")
				decrypted, err := decryptAES(key, aesBeforeDecrypt)
				if err != nil {
					log.Fatalf("something went wrong during decrypt")
				}
				if verbose == "true" {
					fmt.Println(whitespaces + stringArray[0] + " " + decrypted)
				}
				stringArray[1] = decrypted
				tmpYamlText = append(tmpYamlText, whitespaces+strings.Join(stringArray, " "))

			}
		} else {
			if verbose == "true" {
				fmt.Println(eachLn)
			}
			tmpYamlText = append(tmpYamlText, eachLn)
		}
		matchedEnvVariable, _ := regexp.MatchString(env, eachLn)
		if matchedEnvVariable || eachLn == "" {
			envWhiteSpaces = currentWhiteSpaces + 2
			valuesForFormatting = true
		}
	}

	// if already ok, read temp yaml slice and rewrite target yaml file
	if debug != "true" {
		file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed open file: %s", err)
		}
		datawriter := bufio.NewWriter(file)
		for _, data := range tmpYamlText {
			_, _ = datawriter.WriteString(data + "\n")
		}
		err = datawriter.Flush()
		if err != nil {
			return
		}
		err = file.Close()
		if err != nil {
			return
		}
	}
}

func countLeadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func readFile(filename string) (text []string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}
	err = file.Close()
	if err != nil {
		return nil
	}
	return text

}

func encryptAES(password string, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key := make([]byte, 32)
	copy(key, password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	content := []byte(plaintext)
	blockSize := block.BlockSize()
	padding := blockSize - len(content)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	content = append(content, padtext...)

	ciphertext := make([]byte, aes.BlockSize+len(content))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], content)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAES(password string, crypt64 string) (string, error) {
	if crypt64 == "" {
		return "", nil
	}

	key := make([]byte, 32)
	copy(key, password)

	crypt, err := base64.StdEncoding.DecodeString(crypt64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := crypt[:aes.BlockSize]
	crypt = crypt[aes.BlockSize:]
	decrypted := make([]byte, len(crypt))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, crypt)

	return string(decrypted[:len(decrypted)-int(decrypted[len(decrypted)-1])]), nil
}
