package publisher

import (
	"context"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type GeofencePublisher interface {
	PublishAlert(ctx context.Context, alert *domain.GeofenceAlert) error
}
