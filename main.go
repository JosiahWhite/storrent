package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/pflag"

	"github.com/anacrolix/torrent"
)

func main() {
	flagSet := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flagSet.PrintDefaults()
	}
	//configPath := flagSet.StringP("config", "c", "$HOME/.storrent.conf", "config file to use")

	torrentFile := flagSet.StringP("torrent", "t", "", "torrent file")
	dataPath := flagSet.StringP("data", "d", "", "data dir")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			return
		}

		log.Fatalln(err)
	}

	/*
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
	*/

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = *dataPath
	clientConfig.NoDHT = true
	clientConfig.NoUpload = false
	clientConfig.DisableAggressiveUpload = false
	clientConfig.Seed = true
	//clientConfig.Debug = true
	clientConfig.ListenPort = 0
	clientConfig.TorrentPeersLowWater = 500
	clientConfig.TorrentPeersHighWater = 500
	clientConfig.EstablishedConnsPerTorrent = 500
	clientConfig.HalfOpenConnsPerTorrent = 500

	torrentClient, err := torrent.NewClient(clientConfig)
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			log.Printf("close signal received: %+v", <-c)
			torrentClient.Close()

			os.Exit(0)
		}
	}()

	t, err := torrentClient.AddTorrentFromFile(*torrentFile)
	if err != nil {
		log.Fatal(err)
	}

	// temporary
	rs := reflect.ValueOf(t).Elem()
	rf := rs.FieldByName("requestStrategy")
	rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
	rf.SetInt(1)

	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()

	oldRead := int64(0)
	oldWrite := int64(0)

	for {
		torrentStatus := t.Stats()

		readBps := torrentStatus.ConnStats.BytesReadData.Int64() - oldRead
		oldRead = torrentStatus.ConnStats.BytesReadData.Int64()

		writeBps := torrentStatus.ConnStats.BytesWrittenData.Int64() - oldWrite
		oldWrite = torrentStatus.ConnStats.BytesWrittenData.Int64()

		readBps = int64(readBps / 5)
		writeBps = int64(writeBps / 5)

		doneBytes := t.BytesCompleted()
		totalBytes := t.Info().TotalLength()
		doneFloat := float64(doneBytes) / float64(totalBytes)

		fmt.Printf("\033c")
		fmt.Println("{")
		fmt.Println(" local port:", torrentClient.LocalPort())
		fmt.Println(" done:", byteCountBinary(doneBytes))
		fmt.Println(" size:", byteCountBinary(totalBytes))
		fmt.Printf(" Percentage: %f%%\n", doneFloat*100)
		fmt.Println(" Read Bytes:", byteCountBinary(oldRead))
		fmt.Println(" Write Bytes:", byteCountBinary(oldWrite))
		fmt.Println(" Speed:")
		fmt.Println("  Down:", byteCountBinary(readBps))
		fmt.Println("  Up:", byteCountBinary(writeBps))
		fmt.Println(" TotalPeers:", torrentStatus.TotalPeers)
		fmt.Println(" PendingPeers:", torrentStatus.PendingPeers)
		fmt.Println(" ActivePeers:", torrentStatus.ActivePeers)
		fmt.Println(" ConnectedSeeders:", torrentStatus.ConnectedSeeders)
		fmt.Println(" HalfOpenPeers:", torrentStatus.HalfOpenPeers)
		fmt.Println("}")
		start := time.Now()
		t.PieceAvailabilityList()
		elapsed := time.Since(start)

		fmt.Println("piece fetch time:", elapsed)

		time.Sleep(5 * time.Second)
	}
}
