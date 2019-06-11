package main

import (
	"errors"
	"fmt"
	"os/signal"
	"strings"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/dao"
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/WeTrustPlatform/account-indexer/watcher"
	log "github.com/sirupsen/logrus"

	"github.com/WeTrustPlatform/account-indexer/http"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue"
	"github.com/ethereum/go-ethereum/console"
	"github.com/syndtr/goleveldb/leveldb"

	"os"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	app = newApp()

	ipcFlag = cli.StringFlag{
		Name:  "ipc",
		Usage: "ipc file paths separated by ','",
		Value: common.DefaultIpc,
	}
	dbFlag = cli.StringFlag{
		Name:  "db",
		Usage: "leveldb file path",
		Value: common.DefaultDbPath,
	}
	cleanIntervalFlag = cli.Float64Flag{
		Name:  "bci",
		Usage: "block clean interval (int) in minute",
		Value: common.DefaultCleanInterval,
	}
	blockTimeToLiveFlag = cli.Float64Flag{
		Name:  "bttl",
		Usage: "block time to live (int) in hour",
		Value: common.DefaultBlockTTL,
	}
	watcherIntervalFlag = cli.Float64Flag{
		Name:  "w",
		Usage: "watcher interval (int) in minute",
		Value: common.DefaultWatcherInterval,
	}
	oosThresholdFlag = cli.Float64Flag{
		Name:  "oos",
		Usage: "threshold (in second) to consider a node as out of sync",
		Value: common.DefaultOOSThreshold,
	}
	portFlag = cli.IntFlag{
		Name:  "p",
		Usage: "http port number",
		Value: common.DefaultHTTPPort,
	}

	batchFlag = cli.IntFlag{
		Name:  "b",
		Usage: "initial number of batch (1-127)",
		Value: common.DefaultNumBatch,
	}

	indexerFlags = []cli.Flag{
		ipcFlag,
		dbFlag,
		cleanIntervalFlag,
		blockTimeToLiveFlag,
		watcherIntervalFlag,
		oosThresholdFlag,
		portFlag,
		batchFlag,
	}
)

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "indexer"
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = config.GetVersion().Version
	app.Usage = "the indexer for geth"
	return app
}

func setConfig(ctx *cli.Context) {
	clearInterval := ctx.GlobalFloat64(cleanIntervalFlag.Name)
	blockTTL := ctx.GlobalFloat64(blockTimeToLiveFlag.Name)
	watcherInterval := ctx.GlobalFloat64(watcherIntervalFlag.Name)
	oosThreshold := ctx.GlobalFloat64(oosThresholdFlag.Name)
	config := config.GetConfig()
	config.CleanInterval = time.Duration(clearInterval) * time.Minute
	if config.CleanInterval < 1*time.Second {
		panic(fmt.Errorf("CleanInterval of %v is not valid", config.CleanInterval))
	}
	config.BlockTTL = time.Duration(blockTTL) * time.Hour
	if config.BlockTTL < 1*time.Hour {
		panic(fmt.Errorf("BlockTTL of %v is not valid. Should not be less than 1 hour", config.BlockTTL))
	}
	config.WatcherInterval = time.Duration(watcherInterval) * time.Minute
	if config.WatcherInterval < 1*time.Second {
		panic(fmt.Errorf("WatcherInterval of %v is not valid", config.WatcherInterval))
	}
	config.OOSThreshold = time.Duration(oosThreshold) * time.Second
	if config.OOSThreshold < 1*time.Second {
		panic(fmt.Errorf("OOSThreshold of %v is not valid", config.OOSThreshold))
	}

	config.Port = ctx.GlobalInt(portFlag.Name)
	config.NumBatch = ctx.GlobalInt(batchFlag.Name)
	config.StartTime = time.Now()
	// byte range
	if config.NumBatch < 1 || config.NumBatch > 127 {
		panic(errors.New("number of batch should be 1 to 127"))
	}
	log.WithField("config", config).Info("Found configuration")
}

// Entry point
func index(ctx *cli.Context) {
	setConfig(ctx)
	ipcPath := ctx.GlobalString(ipcFlag.Name)
	dbPath := ctx.GlobalString(dbFlag.Name)
	config.GetConfig().DbPath = dbPath

	ipcs := strings.Split(ipcPath, ",")
	service.GetIpcManager().SetIPC(ipcs)
	addressDB, err := leveldb.OpenFile(dbPath+"_address", nil)
	if err != nil {
		panic(errors.New("Can't connect to Address LevelDB. Error: " + err.Error()))
	}
	blockDB, err := leveldb.OpenFile(dbPath+"_block", nil)
	if err != nil {
		panic(errors.New("Can't connect to Block LevelDB. Error: " + err.Error()))
	}
	batchDB, err := leveldb.OpenFile(dbPath+"_batch", nil)
	if err != nil {
		panic(errors.New("Can't connect to Batch LevelDB. Error: " + err.Error()))
	}

	cleanUp := func() {
		addressDB.Close()
		blockDB.Close()
		batchDB.Close()
	}
	defer cleanUp()
	interuptChan := make(chan os.Signal, 1)
	signal.Notify(interuptChan, os.Interrupt)
	go func() {
		<-interuptChan
		cleanUp()
		log.Info("Received interupt signal, cleanup and exit...")
		os.Exit(1)
	}()

	indexRepo := keyvalue.NewKVIndexRepo(dao.NewLevelDbDAO(addressDB), dao.NewLevelDbDAO(blockDB))
	batchRepo := keyvalue.NewKVBatchRepo(dao.NewLevelDbDAO(batchDB))
	idx := indexer.NewIndexer(indexRepo, batchRepo, nil)
	go idx.FirstIndex()
	cleaner := watcher.NewCleaner(indexRepo)
	go cleaner.CleanBlockDB()
	server := http.NewServer(idx)
	server.Start()
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	logInit()
	app.Action = index
	app.Flags = append(app.Flags, indexerFlags...)
	// app.Before
	app.After = func(ctx *cli.Context) error {
		// debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func logInit() {
	// Log as JSON instead of the default ASCII formatter.
	// log.SetFormatter(&log.JSONFormatter{})

	// Only log the warning severity or above.
	logLevel := os.Getenv("INDEXER_LOG_LEVEL")
	logLevelMap := map[string]log.Level{
		"":      log.InfoLevel,
		"panic": log.PanicLevel,
		"fatal": log.FatalLevel,
		"error": log.ErrorLevel,
		"warn":  log.WarnLevel,
		"info":  log.InfoLevel,
		"debug": log.DebugLevel,
		"trace": log.TraceLevel,
	}
	log.SetLevel(logLevelMap[strings.ToLower(logLevel)])
}
