package main

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLoadManifestFile(t *testing.T) {
	var m Manifest
	var err error
	m, err = LoadManifestFile(filepath.Join("testdata", "deployer-1.yml"))
	assert.NoError(t, err)
	p := m.Profile("dev")
	var buf []byte
	buf, err = p.GenerateBuild()
	assert.NoError(t, err)
	t.Log(string(buf))
	buf, err = p.GeneratePackage()
	assert.NoError(t, err)
	t.Log(string(buf))
}
