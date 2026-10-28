package catalog

import (
	"regexp"
	"strings"

	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

var (
	grafanaInterval = regexp.MustCompile(`\$\{?__(?:rate_interval|interval|range|auto)\}?`)
	promql          = parser.NewParser(parser.Options{})
	histogramSeries = map[string]bool{"_bucket": true, "_count": true, "_sum": true}
)

func Check(query artifact.Query) error {
	expr := grafanaInterval.ReplaceAllString(query.Expr, "1m")

	switch query.Language {
	case artifact.PromQL:
		return checkPromQL(expr)
	case artifact.LogQL:
		return checkLogQL(expr)
	default:
		return errors.New("unknown query language %q", query.Language)
	}
}

type labelSet map[string]bool

func setOf(groups ...[]string) labelSet {
	set := labelSet{}

	for _, group := range groups {
		for _, label := range group {
			set[label] = true
		}
	}

	return set
}

func union(sets ...labelSet) labelSet {
	out := labelSet{}

	for _, set := range sets {
		for label := range set {
			out[label] = true
		}
	}

	return out
}

func checkPromQL(input string) error {
	expr, err := promql.ParseExpr(input)
	if err != nil {
		return errors.New("parsing PromQL %q", input).WithCause(err)
	}

	_, err = labelsOf(expr)

	return err
}

func labelsOf(node parser.Expr) (labelSet, error) {
	switch n := node.(type) {
	case *parser.VectorSelector:
		return selectorLabels(n)
	case *parser.MatrixSelector:
		return labelsOf(n.VectorSelector)
	case *parser.SubqueryExpr:
		return labelsOf(n.Expr)
	case *parser.ParenExpr:
		return labelsOf(n.Expr)
	case *parser.UnaryExpr:
		return labelsOf(n.Expr)
	case *parser.StepInvariantExpr:
		return labelsOf(n.Expr)
	case *parser.NumberLiteral, *parser.StringLiteral:
		return labelSet{}, nil
	case *parser.AggregateExpr:
		return aggregateLabels(n)
	case *parser.BinaryExpr:
		return binaryLabels(n)
	case *parser.Call:
		return callLabels(n)
	default:
		return nil, errors.New("unsupported PromQL expression %q", node.String())
	}
}

func selectorLabels(selector *parser.VectorSelector) (labelSet, error) {
	name := selector.Name

	for _, matcher := range selector.LabelMatchers {
		if matcher.Name == model.MetricNameLabel && matcher.Type == labels.MatchEqual {
			name = matcher.Value
		}
	}

	allowed := setOf(targetLabels, []string{config.EnvironmentLabel})

	if name != "" {
		var err error
		if allowed, err = metricLabels(name); err != nil {
			return nil, err
		}
	}

	for _, matcher := range selector.LabelMatchers {
		if !allowed[matcher.Name] {
			if name == "" {
				return nil, errors.New("label %q is not allowed on a selector without a metric name", matcher.Name)
			}

			return nil, errors.New("label %q is not allowed on metric %q", matcher.Name, name)
		}
	}

	return allowed, nil
}

func metricLabels(series string) (labelSet, error) {
	if metric, ok := lookup(series); ok {
		return setOf(targetLabels, sourceLabels[metric.Source], metric.Labels), nil
	}

	for suffix := range histogramSeries {
		metric, ok := lookup(strings.TrimSuffix(series, suffix))
		if !ok || metric.Type != Histogram || !strings.HasSuffix(series, suffix) {
			continue
		}

		allowed := setOf(targetLabels, sourceLabels[metric.Source], metric.Labels)
		if suffix == "_bucket" {
			allowed["le"] = true
		}

		return allowed, nil
	}

	return nil, errors.New("metric %q is not in the catalog", series)
}

func requireCarried(carried labelSet, grouping []string) error {
	for _, label := range grouping {
		if !carried[label] {
			return errors.New("grouping by label %q, which the expression does not carry", label)
		}
	}

	return nil
}

func aggregateLabels(aggregate *parser.AggregateExpr) (labelSet, error) {
	inner, err := labelsOf(aggregate.Expr)
	if err != nil {
		return nil, err
	}

	if aggregate.Param != nil {
		if _, err := labelsOf(aggregate.Param); err != nil {
			return nil, err
		}
	}

	if err := requireCarried(inner, aggregate.Grouping); err != nil {
		return nil, err
	}

	switch {
	case aggregate.Op == parser.TOPK || aggregate.Op == parser.BOTTOMK || aggregate.Op == parser.LIMITK:
		return inner, nil
	case aggregate.Without:
		out := union(inner)
		for _, label := range aggregate.Grouping {
			delete(out, label)
		}

		return out, nil
	default:
		return setOf(aggregate.Grouping), nil
	}
}

func binaryLabels(binary *parser.BinaryExpr) (labelSet, error) {
	lhs, err := labelsOf(binary.LHS)
	if err != nil {
		return nil, err
	}

	rhs, err := labelsOf(binary.RHS)
	if err != nil {
		return nil, err
	}

	carried := union(lhs, rhs)

	if matching := binary.VectorMatching; matching != nil {
		if err := requireCarried(carried, matching.MatchingLabels); err != nil {
			return nil, err
		}

		if err := requireCarried(carried, matching.Include); err != nil {
			return nil, err
		}
	}

	return carried, nil
}

func callLabels(call *parser.Call) (labelSet, error) {
	carried := labelSet{}

	for _, arg := range call.Args {
		argLabels, err := labelsOf(arg)
		if err != nil {
			return nil, err
		}

		carried = union(carried, argLabels)
	}

	if call.Func.Name == "histogram_quantile" {
		delete(carried, "le")
	}

	return carried, nil
}

func checkLogQL(input string) error {
	expr, err := syntax.ParseExpr(input)
	if err != nil {
		return errors.New("parsing LogQL %q", input).WithCause(err)
	}

	allowed := setOf(logLabels, faroLabels())

	var (
		used    []string
		walkErr error
	)

	expr.Walk(func(node syntax.Expr) bool {
		switch n := node.(type) {
		case *syntax.MatchersExpr:
			for _, matcher := range n.Mts {
				used = append(used, matcher.Name)
			}
		case *syntax.LabelFilterExpr:
			used = append(used, n.RequiredLabelNames()...)
		case *syntax.LabelFmtExpr:
			for _, format := range n.Formats {
				allowed[format.Name] = true

				if format.Rename {
					used = append(used, format.Value)
				}
			}
		case *syntax.LogRangeExpr:
			if n.Unwrap != nil {
				used = append(used, n.Unwrap.Identifier)
			}
		case *syntax.RangeAggregationExpr:
			used = append(used, groups(n.Grouping)...)
		case *syntax.VectorAggregationExpr:
			used = append(used, groups(n.Grouping)...)
		case *syntax.LineParserExpr, *syntax.LogfmtParserExpr, *syntax.JSONExpressionParserExpr, *syntax.LogfmtExpressionParserExpr:
			walkErr = errors.New("%q extracts labels the catalog cannot bound", node.String())
		}

		return walkErr == nil
	})

	if walkErr != nil {
		return walkErr
	}

	for _, label := range used {
		if !allowed[label] {
			return errors.New("label %q is not a log label in the catalog", label)
		}
	}

	return nil
}

func groups(grouping *syntax.Grouping) []string {
	if grouping == nil {
		return nil
	}

	return grouping.Groups
}
