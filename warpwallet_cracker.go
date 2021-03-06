package main

import (
  pbkdf2 "github.com/ctz/go-fastpbkdf2"
	"golang.org/x/crypto/scrypt"
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"
	"os"
	"math/rand"
	"github.com/vsergeev/btckeygenie/btckey"
)

var c chan []byte // goroutine channel
const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
func random(r *rand.Rand, n int) string {
    b := make([]byte, n)
	for i := range b {
        b[i] = letterBytes[r.Intn(62)]
    }

    return string(b)
}

func main () {
	r := rand.New(rand.NewSource(time.Now().Unix()))
  c = make(chan []byte)

	var address string
	saltValue := ""

	if len(os.Args) >= 2 {
		address = os.Args[1]
		if len(os.Args) == 3 {
			saltValue = os.Args[2]
		} else {
			saltValue = "";
		}
	} else {
		fmt.Printf("Usage: %s [Address] [Salt - optional]\n\n", os.Args[0])
		os.Exit(0)
	}

	fmt.Printf("Using address \"%s\" and salt \"%s\"\n", address, saltValue)

	tries := 0
	start := time.Now()
	for {
		passphraseValue := random(r, 8)
		result := bruteforce(passphraseValue, saltValue, address);
		if result != "" {
			fmt.Printf("Found! Passphrase %s\n", passphraseValue)
			os.Exit(0)
		} else {
			tries += 1
			fmt.Printf("\rTried %d passphrases in %s [last passphrase: %s]", tries, time.Since(start), passphraseValue)
		}
	}
}

func bruteforce(passphraseValue string, saltValue string, address string) string {
  var priv btckey.PrivateKey
  var err error
  go doScrypt(fmt.Sprint(passphraseValue, "\x01"), fmt.Sprint(saltValue, "\x01"), c)
  go doPbkdf2(fmt.Sprint(passphraseValue, "\x02"), fmt.Sprint(saltValue, "\x02"), c)

  key1, key2 := <-c, <-c

  var result bytes.Buffer
  for i := 0; i < len(key1); i++ {
    result.WriteByte(key1[i] ^ key2[i]) // TODO fastXOR?
  }

	err = priv.FromBytes(result.Bytes())
	if err != nil {
		fmt.Printf("Error importing private key: %s [%s]\n", err, passphraseValue)
		return ""
	}

	if (priv.ToAddressUncompressed() == address) {
		return passphraseValue
	}

	return ""
}

func doScrypt(pass string, salt string, c chan []byte) {
   scryptKey, _ := scrypt.Key([]byte(pass), []byte(salt), 262144, 8, 1, 32)
   c <- scryptKey
}

func doPbkdf2(pass string, salt string, c chan []byte) {
  pbkdf2Key := pbkdf2.Key([]byte(pass), []byte(salt), 65536, 32, sha256.New)
  c <- pbkdf2Key
}
