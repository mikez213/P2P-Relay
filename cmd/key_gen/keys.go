package main

import (
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// go run keys.go -n 3
func main() {
	var n int
	flag.IntVar(&n, "n", 10, "number of keys to be generated")
	flag.Parse()

	for i := 0; i < n; i++ {
		priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			panic(err)
		}

		b, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			panic(err)
		}

		encoded := base64.StdEncoding.EncodeToString(b)
		fmt.Println(encoded)
	}
}
