package main

import (
	"bytes"
	"errors"
	"github.com/guoyk93/tempfile"
	"github.com/guoyk93/tmplfuncs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

const (
	ManifestVersion = 2
)

type Profile struct {
	Profile string                 `yaml:"-"`
	Build   []string               `yaml:"build"`
	Package []string               `yaml:"package"`
	Vars    map[string]interface{} `yaml:"vars"`
}

func (p *Profile) Apply(dp Profile) {
	if len(p.Build) == 0 {
		p.Build = dp.Build
	}
	if len(p.Package) == 0 {
		p.Package = dp.Package
	}
	vars := make(map[string]interface{})
	for k, v := range dp.Vars {
		vars[k] = v
	}
	for k, v := range p.Vars {
		vars[k] = v
	}
	p.Vars = vars
}

func (p *Profile) Render(src string) (out []byte, err error) {
	var tmpl *template.Template
	if tmpl, err = template.New("").
		Option("missingkey=zero").
		Funcs(tmplfuncs.Funcs).Parse(src); err != nil {
		return
	}

	envs := map[string]string{}
	for _, env := range os.Environ() {
		splits := strings.SplitN(env, "=", 2)
		if len(splits) == 2 {
			envs[splits[0]] = splits[1]
		}
	}
	data := map[string]interface{}{
		"Env":     envs,
		"Vars":    p.Vars,
		"Profile": p.Profile,
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, data); err != nil {
		return
	}
	out = buf.Bytes()
	return
}

func (p *Profile) GenerateBuild() ([]byte, error) {
	s := &strings.Builder{}
	s.WriteString("#!/bin/bash\nset -eux\n")
	for _, l := range p.Build {
		s.WriteString(l)
		s.WriteRune('\n')
	}
	return p.Render(s.String())
}

func (p *Profile) GeneratePackage() ([]byte, error) {
	return p.Render(strings.Join(p.Package, "\n"))
}

func (p *Profile) PrintGeneratedContent(name string, content string) {
	sb := &strings.Builder{}
	sb.WriteString(name)
	sb.WriteString(":\n--------------------------------------------------\n")
	sb.WriteString(strings.TrimSpace(content))
	sb.WriteString("\n--------------------------------------------------")
	log.Println(sb.String())
	if strings.Contains(content, "<no value>") {
		log.Println("警告：检查到渲染结果出现 <no value>，请确认：")
		log.Printf("  1. 环境名 %s 是否正确 (环境名可能取自 Jenkins 任务后缀)", p.Profile)
		log.Printf("  2. 环境 %s 的 vars 字段是否缺失某些变量", p.Profile)
	}
}

func (p *Profile) GenerateFiles() (buildFile string, packageFile string, err error) {
	var buf []byte
	if buf, err = p.GenerateBuild(); err != nil {
		return
	}
	p.PrintGeneratedContent("构建脚本", string(buf))
	if buildFile, err = tempfile.WriteFile(buf, "deployer-build", ".sh", true); err != nil {
		return
	}
	if buf, err = p.GeneratePackage(); err != nil {
		return
	}
	p.PrintGeneratedContent("打包脚本", string(buf))
	if packageFile, err = tempfile.WriteFile(buf, "deployer-package", ".dockerfile", false); err != nil {
		return
	}
	return
}

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
