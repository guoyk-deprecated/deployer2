package main

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	ManifestVersion = 2
)

type Manifest struct {
	Version  int                `yaml:"version"`
	Default  Profile            `yaml:"default"`
	Profiles map[string]Profile `yaml:",inline"`
}

func LoadManifestFile(file string) (m Manifest, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(file); err != nil {
		return
	}
	if err = yaml.Unmarshal(buf, &m); err != nil {
		return
	}
	if m.Version != ManifestVersion {
		err = errors.New("描述文件中缺少版本号 version: 2")
		return
	}
	return
}

func (m Manifest) Profile(name string) *Profile {
	p := &Profile{}
	*p = m.Profiles[name]
	p.Profile = name
	p.Apply(m.Default)
	return p
}
