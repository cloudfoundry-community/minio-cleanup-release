package main

import (
	"fmt"
	"log"
	"os"

	"cleaner/cmd"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Version will be updated by the build process
var Version = "0.0.0"

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime | log.LUTC)
	app := kingpin.New("cleaner", "Cleaner will remove old versions of files in a minio server")

	configFile := app.Flag("config-file", "Location of config.toml").Short('c').Default("config.toml").Required().ExistingFile()
	dryRun := app.Flag("dry-run", "If set, will list out files to delete but not actually delete them").Bool()
	version := app.Flag("version", "Return the version number").Short('v').Bool()

	app.Parse(os.Args[1:])

	if *version {
		fmt.Printf("Version: %s\n\n", Version)
		os.Exit(0)
	}

	config, err := cmd.ParseConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Execute(config, *dryRun); err != nil {
		log.Fatal(err)
	}
}
