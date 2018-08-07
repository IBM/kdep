package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	fmt.Printf("%x", sum)
}