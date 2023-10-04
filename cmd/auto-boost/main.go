package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"github.com/google/go-github/v55/github"
	"github.com/sirupsen/logrus"

	utils "github.com/thnxdev/utils"
	"github.com/thnxdev/utils/utils/config"
	"github.com/thnxdev/utils/utils/log"

	_ "github.com/thnxdev/utils/workers/wkr-donate"
	_ "github.com/thnxdev/utils/workers/wkr-entities"
	_ "github.com/thnxdev/utils/workers/wkr-repos"
)

// Populated during build.
var GitCommit string = "dev"

var cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version and exit."`
	Config  kong.ConfigFlag  `short:"C" help:"Config file." placeholder:"FILE" env:"CONFIG_PATH"`

	LogLevel logrus.Level `help:"Log level (${enum})." default:"info" enum:"trace,debug,info,warning,error,fatal,panic" group:"Observability:"`
	LogJSON  bool         `help:"Log in JSON format." group:"Observability:"`

	TdApiUrl             utils.TdApiUrl      `help:"API path for thanks.dev." required:"" env:"TD_API_URL" default:"https://api.thanks.dev"`
	TdApiKey             utils.TdApiKey      `help:"API key for thanks.dev." required:"" env:"TD_API_KEY"`
	GhClassicAccessToken utils.GhAccessToken `help:"GitHub classis access token with admin:org & user scopes." required:"" env:"GH_CLASSIC_ACCESS_TOKEN"`

	Entities []utils.Entity `help:"The GitHub entities to process sponsorships for. First entity in the list is considered DEFAULT." required:""`
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
	kctx.Exit = func(code int) {
		kctx.Exit = os.Exit
		stop()
		os.Exit(code)
	}

	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.Bind(cli.TdApiUrl)
	kctx.Bind(cli.TdApiKey)
	kctx.Bind(cli.GhClassicAccessToken)
	kctx.Bind(cli.Entities)

	logger.Info("Starting")

	_, err := kctx.Call(run)
	kctx.FatalIfErrorf(err)

	logger.Info("Exiting")

	kctx.Exit(0)
}

func run(
	ctx context.Context,
	tdApiUrl utils.TdApiUrl,
	tdApiKey utils.TdApiKey,
	ghAccesstoken utils.GhAccessToken,
	entities []utils.Entity,
) error {
	logger := log.FromContext(ctx)

	gclient := github.NewClient(nil).WithAuthToken(string(ghAccesstoken))

	for nextPage := 0; ; {
		repos, resp, err := gclient.Repositories.List(ctx, "", &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    nextPage,
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get repositories")
		}

		for _, r := range repos {
			var entityName *string
			for _, e := range entities {
				if *r.Owner.Login == string(e) {
					entityName = r.Owner.Login
					break
				}
			}

			// ignore any repo that is not in entity list
			// or is archived or is a fork.
			if entityName == nil ||
				(r.Archived != nil && *r.Archived) ||
				(r.Fork != nil && *r.Fork) {
				continue
			}

			var rank int = 1
			pre := regexp.MustCompile("^(tag-non-production|tag-to-be-production-)")
			are := regexp.MustCompile("^(tag-archived|tag-to-be-archived-|tag-lost-and-found-)")
			for _, topic := range r.Topics {
				if topic == "tag-production" {
					rank = 5
					break
				} else if are.Match([]byte(topic)) {
					rank = 0
					break
				} else if pre.Match([]byte(topic)) {
					rank = 3
					break
				}
			}

			payload := fmt.Sprintf(`{"rank":%d}`, rank)

			url := fmt.Sprintf("%s/v1/api/setting/entity/gh/%s/%s", tdApiUrl, *entityName, *r.Name)
			logger.Infof("updating %s", *r.FullName)

			req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(payload)))
			if err != nil {
				return errors.Wrap(err, "failed to create TD request")
			}
			req.Header.Set("content-type", "application/json")
			req.Header.Set("api-key", string(tdApiKey))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Wrap(err, "failed to update")
			}
			defer resp.Body.Close()
			logger.Info(resp.StatusCode)
		}

		nextPage = resp.NextPage
		if nextPage == 0 {
			break
		}

	}
	return nil
}
