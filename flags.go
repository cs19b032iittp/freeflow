package main

import (
	"flag"
)

type config struct {
	listenHost string
	listenPort int
}

func parseFlags() *config {
	c := &config{}

	flag.StringVar(&c.listenHost, "h", "127.0.0.1", "The bootstrap node host listen address\n")
	flag.IntVar(&c.listenPort, "p", 4001, "node listen port")

	flag.Parse()
	return c
}
