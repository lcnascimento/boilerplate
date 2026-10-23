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

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

const (
	MemberKeyUserID         = "user.id"
	MemberKeyOrganizationID = "organization.id"

	attrFailureReason = "iam.authentication.failure_reason"
	attrPermission    = "iam.authorization.permission"
	attrRole          = "iam.authorization.role"
)

var (
	pkg    = "iam/pkg/actor"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)

	authenticationFailures = metric.MustIntCounter(meter,
		"iam.actor.authentication.failures", "Number of requests the actor middleware rejected, by reason")
	authorizationDenials = metric.MustIntCounter(meter,
		"iam.actor.authorization.denials", "Number of actions denied because the actor's role lacks the permission, by permission and role")
)

func onStarted(ctx context.Context) {
	addZeroToEverySeries(ctx)
}

func addZeroToEverySeries(ctx context.Context) {
	for r := range reasonCount {
		authenticationFailures.Add(ctx, 0, rejection(r))
	}

	for _, role := range roles.All() {
		for _, p := range roles.Permissions() {
			if !role.Grants(p) {
				authorizationDenials.Add(ctx, 0, denial(p, role))
			}
		}
	}
}

func rejection(r reason) otelmetric.AddOption {
	return otelmetric.WithAttributes(attribute.String("reason", r.String()))
}

func denial(p roles.Permission, role roles.Role) otelmetric.AddOption {
	return otelmetric.WithAttributes(attribute.String("permission", p.String()), attribute.String("role", role.String()))
}

func onAuthenticated(ctx context.Context, a *Actor) context.Context {
	return baggage.ContextWithMembers(ctx,
		baggage.NewMember(MemberKeyUserID, a.UserID.String()),
		baggage.NewMember(MemberKeyOrganizationID, a.OrganizationID.String()),
	)
}

func onRejected(ctx context.Context, r reason) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.String(attrFailureReason, r.String()))
	authenticationFailures.Add(ctx, 1, rejection(r))
	logger.Warn(ctx, "request rejected: unauthenticated", log.String(attrFailureReason, r.String()))
}

func onDenied(ctx context.Context, a *Actor, p roles.Permission) {
	permission, role := p.String(), a.Role.String()

	trace.SpanFromContext(ctx).SetAttributes(attribute.String(attrPermission, permission), attribute.String(attrRole, role))
	authorizationDenials.Add(ctx, 1, denial(p, a.Role))
	logger.Warn(ctx, "action denied: unauthorized", log.String(attrPermission, permission), log.String(attrRole, role))
}

func onJWKSRefreshFailed(ctx context.Context, err error) {
	logger.Warn(ctx, "failed to refresh the JWKS", logger.ErrorAttr(err))
}
