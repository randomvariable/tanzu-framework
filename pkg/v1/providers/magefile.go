//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"gopkg.in/yaml.v3"

	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
)

type versions struct {
	Providers []provider `yaml:"providers"`
}

type provider struct {
	Name string `yaml:"name"`
	Source string `yaml:"source"`
	Version string `yaml:"version"`
	ComponentManifest manifest `yaml:"componentManifest"`
}

type manifest struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

func readVersions() (*versions, error) {
	dat, err := os.ReadFile("versions.yaml")
	if err != nil {
		return nil, err
	}
	vers := versions{}
	if err := yaml.Unmarshal(dat, &vers); err != nil {
		return nil, err
	}
	fmt.Println(vers)
	return &vers, nil
}

func tanzufyProvider(provider provider) error {
	fmt.Printf("Tanzufying %s\n", provider.Name)
	dirCmd := exec.Command("mkdir", "-p", filepath.Join(provider.Name, provider.Version))
	fmt.Println("Creating directory: ", dirCmd.String())
	out, err := dirCmd.CombinedOutput()
	if err != nil {
		fmt.Println(out)
		return err
	}
	cmd := exec.Command("ytt",
			"--file", filepath.Join("upstream", provider.Source, provider.ComponentManifest.Source),
			"--file", filepath.Join("tanzu-bases", "components", "patches", "common"),
			"--file", filepath.Join("tanzu-bases", "components", provider.Name, "component-manifest.yaml"),
	)
	fmt.Println("Running:", cmd.String())
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	if provider.ComponentManifest.Target == "" {
		provider.ComponentManifest.Target = provider.ComponentManifest.Source
	}
	if err := os.WriteFile(filepath.Join(provider.Name, provider.Version, provider.ComponentManifest.Target), out, 0644); err != nil {
		return err
	}
	metadata, err := os.ReadFile(filepath.Join("upstream", provider.Source, "metadata.yaml"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(provider.Name, provider.Version, "metadata.yaml"), metadata, 0644); err != nil {
		return err
	}
	return nil
}

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	fmt.Println("Building...")
	vers, err := readVersions()
	if err != nil {
		fmt.Println("something went uh oh")
		return err
	}
	for _, provider := range vers.Providers {
		if err := tanzufyProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	mg.Deps(Build)
	fmt.Println("Installing...")
	return os.Rename("./MyApp", "/usr/bin/MyApp")
}
