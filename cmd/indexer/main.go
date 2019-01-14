package main

import (
	"fmt"
	"log"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/repository/dao"

	"github.com/WeTrustPlatform/account-indexer/http"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/watcher"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/params"
	"github.com/syndtr/goleveldb/leveldb"

	"os"
	"os/user"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	app    = newApp()
	usr, _ = user.Current()

	ipcFlag = cli.StringFlag{
		Name:  "ipc",
		Usage: "ipc file path",
		Value: usr.HomeDir + "/working_dir/geth_private_network_data_dir/geth.ipc",
	}
	dbFlag = cli.StringFlag{
		Name:  "db",
		Usage: "leveldb file path",
		Value: usr.HomeDir + "/working_dir/geth_indexer_leveldb",
	}
	cleanIntervalFlag = cli.IntFlag{
		Name:  "bci",
		Usage: "block clean interval (int) in minute",
		Value: common.DefaultCleanInterval,
	}
	blockTimeToLiveFlag = cli.IntFlag{
		Name:  "bttl",
		Usage: "block time to live (int) in hour",
		Value: common.DefaultBlockTTL,
	}

	indexerFlags = []cli.Flag{
		ipcFlag,
		dbFlag,
		cleanIntervalFlag,
		blockTimeToLiveFlag,
	}
)

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "indexer"
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = params.VersionWithMeta
	app.Usage = "the indexer for geth"
	return app
}

// Entry point
func index(ctx *cli.Context) {
	ipcPath := ctx.GlobalString(ipcFlag.Name)
	dbPath := ctx.GlobalString(dbFlag.Name)
	clearInterval := ctx.GlobalInt(cleanIntervalFlag.Name)
	blockTTL := ctx.GlobalInt(blockTimeToLiveFlag.Name)
	config := common.GetConfig()
	config.CleanInterval = time.Duration(clearInterval) * time.Minute
	config.BlockTTL = time.Duration(blockTTL) * time.Hour
	log.Printf("%v ipcPath=%s \n dbPath=%s\n CleanInterval=%v\n BlockTimeToLive=%v\n",
		time.Now(), ipcPath, dbPath, config.CleanInterval, config.BlockTTL)
	addressDB, err := leveldb.OpenFile(dbPath+"_address", nil)
	if err != nil {
		log.Fatal("Can't connect to Address LevelDB", err)
	}
	defer addressDB.Close()
	blockDB, err := leveldb.OpenFile(dbPath+"_block", nil)
	if err != nil {
		log.Fatal("Can't connect to Block LevelDB", err)
	}
	defer blockDB.Close()
	batchDB, err := leveldb.OpenFile(dbPath+"_batch", nil)
	if err != nil {
		log.Fatal("Can't connect to Batch LevelDB", err)
	}
	defer batchDB.Close()
	indexRepo := repository.NewKVIndexRepo(dao.NewLevelDbDAO(addressDB), dao.NewLevelDbDAO(blockDB))
	batchRepo := repository.NewKVBatchRepo(dao.NewLevelDbDAO(batchDB))
	indexer := indexer.Indexer{
		IpcPath:   ipcPath,
		IndexRepo: indexRepo,
		BatchRepo: batchRepo,
	}
	go indexer.Index()
	cleaner := watcher.NewCleaner(indexRepo)
	go cleaner.CleanBlockDB()
	server := http.NewServer(indexRepo, batchRepo)
	server.Start()
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	app.Action = index
	app.Flags = append(app.Flags, indexerFlags...)
	// app.Before
	app.After = func(ctx *cli.Context) error {
		// debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}
