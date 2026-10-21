package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"
)

var directive = regexp.MustCompile(`(` + strings.Join([]string{
	`//nolint([: ]|$)`,
	`#\s*shellcheck\s+disable=`,
	`(//|/\*)\s*eslint-disable`,
	`//\s*@ts-(ignore|expect-error|nocheck)\b`,
}, "|") + `).*`)

var golangciConfig = regexp.MustCompile(`^\.golangci\.ya?ml$`)

func Find(fsys fs.FS, paths []string) ([]string, error) {
	var found []string

	for _, file := range paths {
		content, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, errors.New("reading %s", file).WithCause(err)
		}

		if bytes.IndexByte(content, 0) >= 0 {
			continue
		}

		for line := range bytes.Lines(content) {
			if match := directive.Find(bytes.TrimRight(line, "\r\n")); match != nil {
				found = append(found, file+": "+string(match))
			}
		}

		if !golangciConfig.MatchString(path.Base(file)) {
			continue
		}

		exclusions, err := golangciExclusions(content)
		if err != nil {
			return nil, errors.New("parsing %s", file).WithCause(err)
		}

		for _, exclusion := range exclusions {
			found = append(found, file+": "+exclusion)
		}
	}

	return found, nil
}

func golangciExclusions(content []byte) ([]string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}

	var found []string

	for _, section := range []string{"linters", "formatters"} {
		for _, list := range []string{"presets", "paths", "rules"} {
			for _, item := range lookup(&doc, section, "exclusions", list) {
				line, err := oneLine(item)
				if err != nil {
					return nil, err
				}

				found = append(found, fmt.Sprintf("%s.exclusions.%s: %s", section, list, line))
			}
		}
	}

	return found, nil
}

func lookup(node *yaml.Node, keys ...string) []*yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}

	for _, key := range keys {
		if node.Kind != yaml.MappingNode {
			return nil
		}

		var next *yaml.Node

		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
			}
		}

		if next == nil {
			return nil
		}

		node = next
	}

	if node.Kind != yaml.SequenceNode {
		return nil
	}

	return node.Content
}

func oneLine(node *yaml.Node) (string, error) {
	flatten(node)

	out, err := yaml.Marshal(node)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func flatten(node *yaml.Node) {
	node.HeadComment, node.LineComment, node.FootComment = "", "", ""

	if node.Kind == yaml.ScalarNode {
		node.Style = 0
	} else {
		node.Style = yaml.FlowStyle
	}

	for _, child := range node.Content {
		flatten(child)
	}
}

func Audit(fsys fs.FS, paths []string, allowlist []byte) (unexpected, stale []string, err error) {
	found, err := Find(fsys, paths)
	if err != nil {
		return nil, nil, err
	}

	allowed := map[string]int{}

	for line := range strings.Lines(string(allowlist)) {
		if entry := strings.TrimSpace(line); entry != "" && !strings.HasPrefix(entry, "#") {
			allowed[entry]++
		}
	}

	for _, suppression := range found {
		if allowed[suppression] > 0 {
			allowed[suppression]--
		} else {
			unexpected = append(unexpected, suppression)
		}
	}

	for entry, count := range allowed {
		for range count {
			stale = append(stale, entry)
		}
	}

	slices.Sort(unexpected)
	slices.Sort(stale)

	return unexpected, stale, nil
}
