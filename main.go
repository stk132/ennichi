package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"sync"
)

var host = kingpin.Flag("host", "fireworq host url").Short('h').Required().String()

func main() {
	kingpin.Parse()
	application := newApp(*host)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := application.run(); err != nil {
			log.Fatal(err)
		}
	}()

	application.root.QueueUpdate(func() {
		application.logger.Info().Msg("application start")
	})
	wg.Wait()
}
