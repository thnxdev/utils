package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	errors "github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/config"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"

	_ "github.com/thnxdev/wkr-gh-sponsor/workers/wkr-donate"
	_ "github.com/thnxdev/wkr-gh-sponsor/workers/wkr-entities"
	_ "github.com/thnxdev/wkr-gh-sponsor/workers/wkr-repos"
)

// Populated during build.
var GitCommit string = "dev"

var cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version and exit."`
	Config  kong.ConfigFlag  `short:"C" help:"Config file." placeholder:"FILE" env:"CONFIG_PATH"`

	LogLevel logrus.Level `help:"Log level (${enum})." default:"info" enum:"trace,debug,info,warning,error,fatal,panic" group:"Observability:"`
	LogJSON  bool         `help:"Log in JSON format." group:"Observability:"`

	DbPath               string                     `help:"Path to db file." required:"" env:"DB_PATH" default:"db.sql"`
	GhClassicAccessToken wkrghsponsor.GhAccessToken `help:"GitHub classis access token with admin:org & user scopes." required:"" env:"GH_CLASSIC_ACCESS_TOKEN"`

	Entities []wkrghsponsor.Entity `help:"The GitHub entities to process sponsorships for. First entity in the list is considered DEFAULT." required:""`

	SponsorAmount wkrghsponsor.SponsorAmount `help:"The amount to donate to each dependency" default:"1"`
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

	wkrs := workers.GetWorkers()
	for name, cfg := range wkrs {
		options = append(options,
			kong.Embed(cfg, fmt.Sprintf(`embed:"" prefix:"%s-" group:"%s"`, name, name)),
			kong.Bind(cfg))
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

	kctx.Bind(db)
	kctx.Bind(cli.GhClassicAccessToken)
	kctx.Bind(cli.Entities)
	kctx.Bind(cli.SponsorAmount)

	workers.Run(ctx, wg, kctx)

	logger.Info("exiting")

	kctx.Exit(0)
}
