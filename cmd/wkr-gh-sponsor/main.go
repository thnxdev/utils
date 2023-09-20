package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	errors "github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"github.com/jpillora/backoff"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/config"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"
	wkrtd "github.com/thnxdev/wkr-gh-sponsor/workers/1-td"
	wkranimate "github.com/thnxdev/wkr-gh-sponsor/workers/2-animate"
	wkrdonate "github.com/thnxdev/wkr-gh-sponsor/workers/3-donate"
)

// Populated during build.
var GitCommit string = "dev"

var cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version and exit."`
	Config  kong.ConfigFlag  `short:"C" help:"Config file." placeholder:"FILE" env:"CONFIG_PATH"`

	LogLevel logrus.Level `help:"Log level (${enum})." default:"info" enum:"trace,debug,info,warning,error,fatal,panic" group:"Observability:"`
	LogJSON  bool         `help:"Log in JSON format." group:"Observability:"`

	DbPath               string                     `help:"Path to db file." required:"" env:"DB_PATH" default:"db.sql"`
	TdApiUrl             wkrghsponsor.TdApiUrl      `help:"thanks.dev API URL." required:"" env:"TD_API_URL" default:"https://api.thanks.dev/v1/deps"`
	TdApiKey             wkrghsponsor.TdApiKey      `help:"thanks.dev API key." required:"" env:"TD_API_KEY"`
	GhClassicAccessToken wkrghsponsor.GhAccessToken `help:"GitHub classis access token with admin:org & user scopes." required:"" env:"GH_CLASSIC_ACCESS_TOKEN"`

	Entities []wkrghsponsor.Entity `help:"The GitHub entities to process sponsorships for. First entity in the list is considered DEFAULT." required:""`
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

	cleanup := Cleanup{}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	wg, ctx := errgroup.WithContext(ctx)
	kctx.Exit = func(code int) {
		kctx.Exit = os.Exit
		stop()
		err := wg.Wait()
		for _, fn := range cleanup {
			err := fn()
			if err != nil {
				kctx.Errorf("wkr-gh-sponsor: error: shutdown failed: %v", err)
			}
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			kctx.FatalIfErrorf(err)
		}
		os.Exit(code)
	}

	// Open dbfile
	db, err := database.Open(ctx, cli.DbPath)
	kctx.FatalIfErrorf(err)

	kctx.Bind(db)
	kctx.Bind(cli.TdApiUrl)
	kctx.Bind(cli.TdApiKey)
	kctx.Bind(cli.GhClassicAccessToken)
	kctx.Bind(cli.Entities)

	// Start the workers
	wkrs := []struct {
		name         string
		fn           any
		workInterval time.Duration
	}{
		{
			name:         "wkr-td",
			fn:           wkrtd.New,
			workInterval: time.Hour * 3, // 4 times a day
		},
		{
			name:         "wkr-animate",
			fn:           wkranimate.New,
			workInterval: time.Minute, // once a minute
		},
		{
			name:         "wkr-donate",
			fn:           wkrdonate.New,
			workInterval: time.Minute, // once a minute
		},
	}
	for _, w := range wkrs {
		w := w

		logger := logger.WithField("name", w.name)
		ctx := log.LoggerContext(ctx, logger)

		out, err := kctx.Call(w.fn)
		if err != nil {
			logger.WithError(err).Error("failed to start")
		}

		wkr := out[0].(workers.Worker)

		wg.Go(func() error {
			logger.Info("Starting")
			retry := &backoff.Backoff{
				Min:    w.workInterval,
				Factor: 1.1,
				Jitter: true,
				Max:    w.workInterval * 4,
			}
			for {
				logger.Info("Running")

				// default loop interval is once an hour
				// but if the worker returns an error we'll
				// do a backoff retry in one minute up to 4 mins
				delay := w.workInterval

				err := wkr(ctx)
				if err != nil {
					delay = retry.Duration()
				} else {
					retry.Reset()
				}

				select {
				case <-ctx.Done():
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return ctx.Err()
				case <-time.After(delay):
				}
			}
		})
	}

	err = wg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		kctx.FatalIfErrorf(err)
	}

	kctx.Exit(0)
}

// Cleanup is a convenience type for registering cleanup functions.
type Cleanup []func() error

func (c *Cleanup) Add(f func() error) { *c = append(*c, f) }
