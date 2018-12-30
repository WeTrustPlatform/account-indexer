package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/params"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tuyennhv/geth-indexer/indexer"

	"os"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	app     = newApp()
	ipcFlag = cli.StringFlag{
		Name:  "ipc",
		Usage: "ipc file path",
		Value: "/Users/tuyennguyen/working_dir/geth_private_network_data_dir/geth.ipc",
	}
	dbFlag = cli.StringFlag{
		Name:  "db",
		Usage: "leveldb file path",
		Value: "/Users/tuyennguyen/working_dir/geth_indexer_leveldb",
	}

	indexerFlags = []cli.Flag{
		ipcFlag,
		dbFlag,
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
	fmt.Println(fmt.Sprintf("ipcPath=%s \n dbPath=%s", ipcPath, dbPath))
	addressDB, err := leveldb.OpenFile(dbPath+"_address", nil)
	if err != nil {
		log.Fatal("Can't connect to LevelDB", err)
	}
	defer addressDB.Close()
	blockDB, err := leveldb.OpenFile(dbPath+"_block", nil)
	if err != nil {
		log.Fatal("Can't connect to LevelDB", err)
	}
	defer blockDB.Close()

	// fetcher, err := indexer.NewChainFetch(ipcPath)
	// if err != nil {
	// 	log.Fatal("Can't connect to IPC server", err)
	// 	return
	// }

	indexer := indexer.Indexer{
		// Fetcher: fetcher,
		IpcPath: ipcPath,
		Repo:    indexer.NewLevelDBRepo(addressDB, blockDB),
		// Repo: nil,
	}
	// indexer.RealtimeIndex()
	indexer.IndexFromGenesis()
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
