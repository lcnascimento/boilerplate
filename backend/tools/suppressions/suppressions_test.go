package main

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind_CommentDirectives(t *testing.T) {
	fsys := fstest.MapFS{
		"a/fixture.go": {Data: []byte("package a\n\nvar n = rand.IntN(9) //nolint:gosec // test data\n")},
	}

	found, err := Find(fsys, []string{"a/fixture.go"})
	require.NoError(t, err)

	assert.Equal(t, []string{"a/fixture.go: //nolint:gosec // test data"}, found)
}

func TestFind_EachDirectiveKind(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"shellcheck", "  # shellcheck disable=SC2086", []string{"f: # shellcheck disable=SC2086"}},
		{"eslint next line", "// eslint-disable-next-line no-console -- debug", []string{"f: // eslint-disable-next-line no-console -- debug"}},
		{"eslint block", "/* eslint-disable react/no-danger */", []string{"f: /* eslint-disable react/no-danger */"}},
		{"ts expect error", "  // @ts-expect-error the types lag behind", []string{"f: // @ts-expect-error the types lag behind"}},
		{"ts ignore", "//@ts-ignore", []string{"f: //@ts-ignore"}},
		{"ts nocheck", "// @ts-nocheck", []string{"f: // @ts-nocheck"}},
		{"linter named in a list", "    - nolintlint", nil},
		{"prose mention", "Enable eslint-disable reporting and shellcheck.", nil},
		{"code span in prose", "todo `//nolint` nomeie o linter", nil},
		{"bare nolint", "x := 1 //nolint", []string{"f: //nolint"}},
		{"nolint for every linter with a reason", "x := 1 //nolint // generated", []string{"f: //nolint // generated"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Find(fstest.MapFS{"f": {Data: []byte(tt.line + "\n")}}, []string{"f"})
			require.NoError(t, err)

			assert.Equal(t, tt.want, found)
		})
	}
}

func TestFind_SkipsBinaryFiles(t *testing.T) {
	fsys := fstest.MapFS{"logo.png": {Data: []byte("\x89PNG\x00//nolint\n")}}

	found, err := Find(fsys, []string{"logo.png"})
	require.NoError(t, err)

	assert.Empty(t, found)
}

func TestFind_GolangciExclusions(t *testing.T) {
	config := `version: "2"
linters:
  enable:
    - nolintlint
  exclusions:
    presets:
      - comments
    paths:
      - generated/
    rules:
      # Tests may be long.
      - path: _test\.go
        linters:
          - funlen
          - lll
      - path: dashboards/
        linters: [staticcheck]
        text: "SA1019: .* is deprecated"
formatters:
  exclusions:
    paths:
      - vendor/
`
	fsys := fstest.MapFS{"backend/.golangci.yaml": {Data: []byte(config)}}

	found, err := Find(fsys, []string{"backend/.golangci.yaml"})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"backend/.golangci.yaml: linters.exclusions.presets: comments",
		"backend/.golangci.yaml: linters.exclusions.paths: generated/",
		`backend/.golangci.yaml: linters.exclusions.rules: {path: _test\.go, linters: [funlen, lll]}`,
		`backend/.golangci.yaml: linters.exclusions.rules: {path: dashboards/, linters: [staticcheck], text: 'SA1019: .* is deprecated'}`,
		"backend/.golangci.yaml: formatters.exclusions.paths: vendor/",
	}, found)
}

func TestAudit(t *testing.T) {
	fsys := fstest.MapFS{
		"a.go": {Data: []byte("type T struct {\n" +
			"\tA string `json:\"refId\"` //nolint:tagliatelle // Grafana's schema.\n" +
			"\tB string `json:\"orgId\"` //nolint:tagliatelle // Grafana's schema.\n" +
			"\tC int //nolint:mnd\n" +
			"}\n")},
	}
	paths := []string{"a.go"}

	tests := []struct {
		name           string
		allowlist      string
		wantUnexpected []string
		wantStale      []string
	}{
		{
			name: "every suppression allowed",
			allowlist: "# Reviewed suppressions.\n\n" +
				"a.go: //nolint:tagliatelle // Grafana's schema.\n" +
				"a.go: //nolint:tagliatelle // Grafana's schema.\n" +
				"a.go: //nolint:mnd\n",
		},
		{
			name:           "a duplicate needs its own entry",
			allowlist:      "a.go: //nolint:tagliatelle // Grafana's schema.\na.go: //nolint:mnd\n",
			wantUnexpected: []string{"a.go: //nolint:tagliatelle // Grafana's schema."},
		},
		{
			name: "an entry with no suppression is stale",
			allowlist: "a.go: //nolint:tagliatelle // Grafana's schema.\n" +
				"a.go: //nolint:tagliatelle // Grafana's schema.\n" +
				"a.go: //nolint:mnd\n" +
				"b.go: //nolint:gosec // gone\n",
			wantStale: []string{"b.go: //nolint:gosec // gone"},
		},
		{
			name:           "empty allowlist",
			wantUnexpected: []string{"a.go: //nolint:mnd", "a.go: //nolint:tagliatelle // Grafana's schema.", "a.go: //nolint:tagliatelle // Grafana's schema."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unexpected, stale, err := Audit(fsys, paths, []byte(tt.allowlist))
			require.NoError(t, err)

			assert.Equal(t, tt.wantUnexpected, unexpected)
			assert.Equal(t, tt.wantStale, stale)
		})
	}
}
