package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
)

var host = kingpin.Flag("host", "fireworq host url").Short('h').Required().String()

func main() {
	kingpin.Parse()
	application := newApp(*host)
	if err := application.run(); err != nil {
		log.Fatal(err)
	}
}
