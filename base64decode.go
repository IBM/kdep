package main

import (
	"encoding/base64"
	"os"
	"io/ioutil"
)

func main() {
	data, err := ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, os.Stdin))
	if err != nil {
		panic(err)
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		panic(err)
	}
}
