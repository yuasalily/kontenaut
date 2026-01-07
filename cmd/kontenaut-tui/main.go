package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	cli, err := parseFlags(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Fatal(err)
	}

	// Resolve runtime options with precedence:
	// defaults < config file < env vars < CLI flags
	opts, err := resolveOptions(cli, os.LookupEnv)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(opts); err != nil {
		log.Fatal(err)
	}
}
