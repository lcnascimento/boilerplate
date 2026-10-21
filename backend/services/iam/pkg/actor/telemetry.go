package actor

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/go-kit/o11y/baggage"
	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"
)

const (
	MemberKeyUserID         = "user.id"
	MemberKeyOrganizationID = "organization.id"

	attrFailureReason = "iam.authentication.failure_reason"
)

var (
	pkg    = "iam/pkg/actor"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)

	authenticationFailures = metric.MustIntCounter(meter,
		"iam.actor.authentication.failures", "Number of requests the actor middleware rejected, by reason")
)

func onAuthenticated(ctx context.Context, a *Actor) context.Context {
	return baggage.ContextWithMembers(ctx,
		baggage.NewMember(MemberKeyUserID, a.UserID.String()),
		baggage.NewMember(MemberKeyOrganizationID, a.OrganizationID.String()),
	)
}

func onRejected(ctx context.Context, r reason) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.String(attrFailureReason, string(r)))
	authenticationFailures.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("reason", string(r))))
	logger.Warn(ctx, "request rejected: unauthenticated", log.String(attrFailureReason, string(r)))
}

func onJWKSRefreshFailed(ctx context.Context, err error) {
	logger.Warn(ctx, "failed to refresh the JWKS", logger.ErrorAttr(err))
}
