//nolint:forbidigo
package main

import (
	"encoding/hex"
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
)

var (
	Red   = "\033[31m"
	Green = "\033[32m"
	Reset = "\033[0m"
)

func main() {
	var serialized string
	flag.StringVar(&serialized, "serialized", "", "serialized notifier object in hexadecimal format")
	var key string
	flag.StringVar(&key, "key", "", "base 64 encoded 256-bit AES encryption key")
	flag.Parse()

	fmt.Print("\n\n\n")
	fmt.Println("-------------------------------------------------------------------------------------------------")
	fmt.Println()
	// s := "0a2463363737376134392d316166622d346539372d623839662d34383737613734323361646512185465737420696e746567726174696f6e206d61696c6465761a05656d61696c221668747470733a2f2f6c6f63616c686f73743a383030304a147465737475736572406d61696c6465762e636f6d5a520a216d61696c6465762d736d74702d736572766963652e6d61696c6465763a3130323512147465737475736572406d61696c6465762e636f6d1a0561646d696e28013a0e56756c6e207265706f7274696e679a01346f716e7a63355a764258676452435163664435446463723035676d6d61367a6361376e4566364f674a744b74314933496366553d"
	if serialized == "" {
		fmt.Println(Red + "Serialized notifier string is required" + Reset)
		flag.Usage()
		os.Exit(1)
	}

	data, err := hex.DecodeString(serialized)
	// assert.NoError(t, err)
	if err != nil {
		fmt.Printf(Red+"Error decoding serialized notifier string to bytes: %v, \n"+Reset, err)
		os.Exit(1)
	}

	notifier := &storage.Notifier{}
	err = notifier.UnmarshalVT(data)
	// assert.NoError(t, err)
	if err != nil {
		fmt.Printf(Red+"Error unmarshalling serialized notifier bytes: %v \n"+Reset, err)
		os.Exit(1)
	}

	fmt.Printf(Green+"Encrypted notifier secret: %s \n"+Reset, notifier.GetNotifierSecret())

	if notifier.GetNotifierSecret() != "" && key != "" {
		codec := cryptocodec.NewGCMCryptoCodec()
		// key := "QUVTMjU2S2V5LTMtMzJDaGFyYWN0ZXJzMTIzNDU2Nzg="
		decryptedText, err := codec.Decrypt(key, notifier.GetNotifierSecret())
		// assert.NoError(t, err)
		if err != nil {
			fmt.Printf(Red+"Error decrypting notifier secret: %v \n"+Reset, err)
			os.Exit(1)
		}
		fmt.Printf(Green+"Decrypted notifier secret: %s \n"+Reset, decryptedText)
	}
	fmt.Println()
	fmt.Println("-------------------------------------------------------------------------------------------------")
	fmt.Print("\n\n\n")
}
