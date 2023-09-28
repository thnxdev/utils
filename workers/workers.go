package workers

import (
	"context"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"github.com/jpillora/backoff"
	"github.com/thnxdev/utils/utils/log"
	"golang.org/x/sync/errgroup"
)

var ErrDone = errors.New("worker finished")

type Config struct {
	Disabled     bool          `help:"Flag to disable worker" default:"false"`
	WorkInterval time.Duration `help:"Sleep duration per run while processing" default:"10s"`
}

type Worker func(context.Context) (bool, error)

type factory struct {
	name string
	fn   any
	cfg  *Config
}

var factories []factory

func Register(fn any) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get caller")
	}
	if typ := reflect.TypeOf(fn); typ.Kind() != reflect.Func {
		panic("factory must be a function but is " + typ.String())
	}
	name := strings.Split(path.Base(runtime.FuncForPC(pc).Name()), ".")[0]
	factories = append(factories, factory{name, fn, &Config{}})
}

func GetWorkers() map[string]*Config {
	ret := map[string]*Config{}
	for _, w := range factories {
		ret[w.name] = w.cfg
	}
	return ret
}

func Run(ctx context.Context, wg *errgroup.Group, kctx *kong.Context) {
	logger := log.FromContext(ctx)

	for _, w := range factories {
		w := w

		logger := logger.WithField("name", w.name)
		if w.cfg.Disabled {
			logger.Info("Not starting – disabled")
			continue
		}

		ctx := log.LoggerContext(ctx, logger)

		out, err := kctx.Call(w.fn)
		if err != nil {
			logger.WithError(err).Error("failed to start")
		}

		wkr := out[0].(Worker)

		wg.Go(func() error {
			logger.Info("Starting")
			retry := &backoff.Backoff{
				Min:    w.cfg.WorkInterval,
				Factor: 1.1,
				Jitter: true,
				Max:    w.cfg.WorkInterval * 4,
			}
			for {
				logger.Info("Running")

				delay := w.cfg.WorkInterval

				hasMore, err := wkr(ctx)
				if err != nil {
					if errors.Is(err, ErrDone) {
						logger.Info("Finished")
						return nil
					}
					logger.WithError(err).Error("error encountered")
					delay = retry.Duration()
				} else if hasMore {
					continue
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

	err := wg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		kctx.FatalIfErrorf(err)
	}

	logger.Info("Exiting")
}
