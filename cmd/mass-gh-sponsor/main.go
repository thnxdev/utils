package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	errors "github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/utils/config"
	"github.com/thnxdev/utils/utils/log"

	animaterepos "github.com/thnxdev/utils/commands/animate-repos"
	dlrepos "github.com/thnxdev/utils/commands/dl-repos"
	"github.com/thnxdev/utils/commands/donate"
	importcsv "github.com/thnxdev/utils/commands/import-csv"
)

// Populated during build.
var GitCommit string = "dev"

var cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version and exit."`
	Config  kong.ConfigFlag  `short:"C" help:"Config file." placeholder:"FILE" env:"CONFIG_PATH"`

	LogLevel logrus.Level `help:"Log level (${enum})." default:"info" enum:"trace,debug,info,warning,error,fatal,panic" group:"Observability:"`
	LogJSON  bool         `help:"Log in JSON format." group:"Observability:"`

	DbPath string `help:"Path to db file." required:"" env:"DB_PATH" default:"db.sql"`

	ImportCsv    importcsv.CmdImportCsv       `cmd:"" help:"Import list of donations from csv file."`
	DlRepos      dlrepos.CmdDlRepos           `cmd:"" help:"Import the user's github repos."`
	AnimateRepos animaterepos.CmdAnimateRepos `cmd:"" help:"Animate the sponsorable dependencies for each repo."`
	Donate       donate.CmdDonate             `cmd:"" help:"Create the require GitHub sponsorships."`
}

func main() {
	options := []kong.Option{
		kong.Configuration(config.CreateLoader),
		kong.HelpOptions{Compact: true},
		kong.AutoGroup(func(parent kong.Visitable, flag *kong.Flag) *kong.Group {
			node, ok := parent.(*kong.Command)
			if !ok {
				return nil
			}
			return &kong.Group{Key: node.Name, Title: "Command flags:"}
		}),
		kong.Vars{
			"version": GitCommit,
		},
	}

	kctx := kong.Parse(&cli, options...)

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(cli.LogLevel)
	if cli.LogJSON {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	ctx := log.LoggerContext(context.Background(), logger)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	wg, ctx := errgroup.WithContext(ctx)
	kctx.Exit = func(code int) {
		kctx.Exit = os.Exit
		stop()
		err := wg.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			kctx.FatalIfErrorf(err)
		}
		os.Exit(code)
	}

	// Open dbfile
	db, err := database.Open(ctx, cli.DbPath)
	kctx.FatalIfErrorf(err)

	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.Bind(db)

	err = kctx.Run()
	kctx.FatalIfErrorf(err)

	kctx.Exit(0)
}
