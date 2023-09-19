package workers

import "context"

type Worker func(context.Context) error
