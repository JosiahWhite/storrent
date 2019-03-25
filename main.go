package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/boltdb/bolt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	flagSet := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flagSet.PrintDefaults()
	}
	configPath := flagSet.StringP("config", "c", "$HOME/.storrent.conf", "config file to use")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			return
		}

		log.Fatalln(err)
	}

	viper.SetConfigType("toml")
	viper.SetConfigFile(*configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("Failed reading config file:", err)
	}

	// TODO: Set defaults for every option where it makes sense

	dbPath := viper.GetString("sessionDB")
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalln("Failed to open session database:", err)
	}
	defer db.Close()

}
