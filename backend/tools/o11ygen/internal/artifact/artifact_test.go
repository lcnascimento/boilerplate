package artifact_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func kindOf(files ...artifact.File) artifact.Kind {
	return artifact.Kind{
		Dir:   "rules",
		Glob:  "*.yaml",
		Build: func([]service.Service) ([]artifact.File, error) { return files, nil },
	}
}

func TestGenerate_RefusesContentCarryingAnExampleFence(t *testing.T) {
	root := t.TempDir()
	content := []byte("groups:\n# example" + ":begin\n")

	err := artifact.Generate(root, nil, kindOf(artifact.File{Name: "notes.yaml", Content: content}))

	require.Error(t, err)
	assert.NoFileExists(t, root+"/rules/notes.yaml")
}

func TestGenerate_RefusesAFileItsKindWouldNotCleanUp(t *testing.T) {
	err := artifact.Generate(t.TempDir(), nil, kindOf(artifact.File{Name: "notes.json", Content: []byte("{}")}))

	require.Error(t, err)
}
