package main

import (
	"fmt"

	"github.com/SanTiwari07/NDVI_satellite/internal/crypto"
)

func main() {
	for _, pw := range []string{"password123", "aA1!£€ 😀"} {
		h, err := crypto.GeneratePassword(pw)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\t%s\n", pw, h)
	}
}
