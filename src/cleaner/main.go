package main

import (
	"log"
	"os"

	"cleaner/cmd"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Version will be the commit hash where the final release bumps
var Version = "2bca79219258c09837a3c60c504584f44d72405e"

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime | log.LUTC)
	log.SetOutput(os.Stdout)
	app := kingpin.New("cleaner", "Cleaner will remove old versions of files in a minio server")
	app.Version(Version)

	configFile := app.Flag("config-file", "Location of config.toml").Short('c').Required().ExistingFile()
	dryRun := app.Flag("dry-run", "If set, will list out files to delete but not actually delete them").Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	config, err := cmd.ParseConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Execute(config, *dryRun); err != nil {
		log.Fatal(err)
	}
}
