package main

import (
	//"crypto"
	//"crypto/aes"
	//"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	//"io"
	"os"
	"io/ioutil"
	"crypto"
	"encoding/pem"
	"crypto/x509"
	"crypto/rsa"
)

func verify(sigString string, rsaPubKey rsa.PublicKey) {
	message := []byte("This is a genuine request!")
	hashed := sha256.Sum256(message)
	signature, _ := hex.DecodeString(sigString)
	err := rsa.VerifyPKCS1v15(&rsaPubKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		fmt.Println("Error")
	}
	fmt.Println("Valid signature")
}

func main() {

	privateKeyFile := os.Args[1]
	pemString, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		fmt.Println(err)
	} else {
		block, _ := pem.Decode([]byte(pemString))
		key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
		rng := rand.Reader
		message := []byte("This is a genuine request!")
		hashed := sha256.Sum256(message)
		signature, err := rsa.SignPKCS1v15(rng, key, crypto.SHA256, hashed[:])
		if err != nil {
			fmt.Println(err)
		} else {
			sigString := hex.EncodeToString(signature)
			fmt.Println(sigString)
			f, err := os.Create("sig")
			if err == nil {
				f.WriteString(sigString)
			}
			//verify(sigString, key.PublicKey)
		}

	}
}
