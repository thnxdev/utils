package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
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

	TdApiUrl utils.TdApiUrl `help:"API path for thanks.dev." required:"" env:"TD_API_URL" default:"https://api.thanks.dev"`
	TdApiKey utils.TdApiKey `help:"API key for thanks.dev." required:"" env:"TD_API_KEY"`

	Outpath string `help:"Path to the output export file." required:"" default:"out.csv"`
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
	kctx.Bind(cli.Outpath)

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
	outpath string,
) error {
	dependencies, err := getDepsFundable(ctx, tdApiUrl, tdApiKey)
	if err != nil {
		return err
	}

	inclusions, err := getInclusions(ctx, tdApiUrl, tdApiKey)
	if err != nil {
		return err
	}

	incIndex := map[string]bool{}
	for _, _inc := range inclusions {
		inc := _inc.([]any)
		name := inc[0].(string)
		wantsFunding := inc[2].(bool)
		isOnTd := inc[3].(bool)
		if wantsFunding || isOnTd {
			incIndex[name] = true
		}
	}

	records := [][]string{}
	for _, _dep := range dependencies {
		dep := _dep.([]any)
		name := dep[0].(string)

		isIncluded, isOnTd := "false", "false"
		if _, ok := incIndex[name]; ok {
			isIncluded = "true"
		}

		if dep[1].(bool) {
			isOnTd = "true"
		}

		dependeeRepos := []string{}
		for _, r := range dep[2].([]any) {
			dependeeRepos = append(dependeeRepos, r.(string))
		}
		dependerEntities := []string{}
		for _, r := range dep[3].([]any) {
			dependerEntities = append(dependerEntities, r.(string))
		}

		records = append(records, []string{
			name,
			isIncluded,
			isOnTd,
			strings.Join(dependeeRepos, ","),
			strings.Join(dependerEntities, ","),
		})
	}

	f, err := os.Create(outpath)
	if err != nil {
		return err
	}

	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"name", "isIncluded", "isOnTd", "repos", "entities"})
	w.WriteAll(records)

	return nil
}

func getDepsFundable(
	ctx context.Context,
	tdApiUrl utils.TdApiUrl,
	tdApiKey utils.TdApiKey,
) ([]any, error) {
	url := fmt.Sprintf("%s/v1/api/deps/fundable", tdApiUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TD request")
	}
	req.Header.Set("api-key", string(tdApiKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update")
	}
	defer resp.Body.Close()

	var res struct {
		Dependencies []any `json:"dependencies"`
	}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	return res.Dependencies, nil
}

func getInclusions(
	ctx context.Context,
	tdApiUrl utils.TdApiUrl,
	tdApiKey utils.TdApiKey,
) ([]any, error) {
	url := fmt.Sprintf("%s/v1/api/setting/inclusions", tdApiUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TD request")
	}
	req.Header.Set("api-key", string(tdApiKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update")
	}
	defer resp.Body.Close()

	var res struct {
		Inclusions []any `json:"inclusions"`
	}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	return res.Inclusions, nil
}
