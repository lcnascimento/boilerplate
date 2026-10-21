package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/lcnascimento/go-kit/errors"
)

const (
	allowlistFile = ".lint-suppressions"
	selfDir       = "backend/tools/suppressions/"
)

func main() {
	root := flag.String("root", ".", "repository root")

	flag.Parse()

	clean, err := run(context.Background(), *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "suppressions:", strings.Join(append([]string{err.Error()}, errors.Reasons(err)...), ": "), errors.Attributes(err))
		os.Exit(1)
	}

	if !clean {
		os.Exit(1)
	}
}

func run(ctx context.Context, root string) (bool, error) {
	fsys := os.DirFS(root)

	paths, err := listFiles(ctx, root, fsys)
	if err != nil {
		return false, err
	}

	allowlist, err := fs.ReadFile(fsys, allowlistFile)
	if err != nil {
		return false, errors.New("failed to read the allowlist").WithAttribute("path", allowlistFile).WithCause(err)
	}

	unexpected, stale, err := audit(fsys, paths, allowlist)
	if err != nil {
		return false, err
	}

	report(unexpected, "lint suppression(s) missing from "+allowlistFile+
		": remove them and fix the code, or add these lines there for review")
	report(stale, "entry(ies) in "+allowlistFile+" match no suppression: remove them")

	return len(unexpected) == 0 && len(stale) == 0, nil
}

func listFiles(ctx context.Context, root string, fsys fs.FS) ([]string, error) {
	git := exec.CommandContext(ctx, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	git.Dir = root

	out, err := git.Output()
	if err != nil {
		return nil, errors.New("failed to list the repository files").WithAttribute("root", root).WithCause(err)
	}

	var paths []string

	for _, raw := range bytes.Split(bytes.TrimSuffix(out, []byte{0}), []byte{0}) {
		path := string(raw)
		if path == allowlistFile || strings.HasPrefix(path, selfDir) {
			continue
		}

		info, err := fs.Lstat(fsys, path)
		if errors.Is(err, fs.ErrNotExist) || err == nil && !info.Mode().IsRegular() {
			continue
		}

		if err != nil {
			return nil, errors.New("failed to stat a file").WithAttribute("path", path).WithCause(err)
		}

		paths = append(paths, path)
	}

	return paths, nil
}

func report(entries []string, problem string) {
	if len(entries) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "suppressions: %d %s\n", len(entries), problem)

	for _, entry := range entries {
		fmt.Fprintln(os.Stderr, entry)
	}
}
