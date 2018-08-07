package main

import (
	"encoding/base64"
	"os"
	"io/ioutil"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	_, err = encoder.Write(data)
	if err != nil {
		panic(err)
	}
	err = encoder.Close()
	if err != nil {
		panic(err)
	}
}
