package main

import (
	"fmt"
	"log"
	"time"

	"github.com/WeTrustPlatform/account-indexer/http"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository"
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
	fmt.Println(fmt.Sprintf("%v ipcPath=%s \n dbPath=%s\n", time.Now(), ipcPath, dbPath))
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
	batchDB, err := leveldb.OpenFile(dbPath+"_batch", nil)
	if err != nil {
		log.Fatal("Can't connect to LevelDB", err)
	}
	defer batchDB.Close()
	repo := repository.NewLevelDBRepo(addressDB, blockDB, batchDB)
	indexer := indexer.Indexer{
		IpcPath: ipcPath,
		Repo:    repo,
	}
	go indexer.Index()
	server := http.NewServer(repo)
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
