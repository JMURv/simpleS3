package cleaner

import "context"

type Cleaner interface {
	Clean(ctx context.Context)
}
