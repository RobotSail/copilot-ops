package config

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/redhat-et/copilot-ops/pkg/ai"
	"github.com/redhat-et/copilot-ops/pkg/ai/bloom"
	"github.com/redhat-et/copilot-ops/pkg/ai/gpt3"
	"github.com/redhat-et/copilot-ops/pkg/ai/gptj"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	ConfigName      = ".copilot-ops"
	ConfigFile      = ".copilot-ops.yaml"
	ConfigFileLocal = ".copilot-ops.local"
)

// Config Defines the struct into which the config-file will be parsed.
type Config struct {
	Filesets []Filesets `json:"filesets,omitempty" yaml:"filesets,omitempty"`
	// Backend Defines which AI backend should be used in order to generate completions.
	// Valid models include: gpt-3, gpt-j, opt, and bloom.
	Backend ai.Backend `json:"backend" yaml:"backend,omitempty"`
	// GPT3 Defines the settings necessary for the GPT3 GPT-3 backend.
	// FIXME: rename to GPT-3
	GPT3 *gpt3.Config `json:"gpt3,omitempty" yaml:"gpt3,omitempty"`
	// GPTJ Defines the configuration options for using GPT-J.
	GPTJ *gptj.Config `json:"gptj,omitempty" yaml:"gptj,omitempty"`
	// BLOOM Defines the configuration for using BLOOM.
	BLOOM *bloom.Config `json:"bloom,omitempty" yaml:"bloom,omitempty"`
}

type Filesets struct {
	Name  string   `json:"name" yaml:"name"`
	Files []string `json:"files" yaml:"files"`
}

// OpenAI Defines the settings for accessing and using OpenAI's tooling.
// GPTJ Defines the structure required for configuring GPT-J.
type GPTJ struct {
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}

const EnvPrefix = "COPILOT_OPS"

// we'll just use the defaults and continue without error.
// Errors here might return if the file exists but is invalid.
func (c *Config) Load(cmd *cobra.Command) error {
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	// set defaults per engine, users can override
	viper.SetDefault("gpt3", gpt3.DefaultConfig)
	viper.SetDefault("gptj", gptj.DefaultConfig)
	viper.SetDefault("bloom", bloom.DefaultConfig)

	// paths to look for the config file in
	viper.AddConfigPath("/etc")
	viper.AddConfigPath("${HOME}")
	// viper.AddConfigPath("..") // parent? grandparent? grandgrandparent?
	viper.AddConfigPath(".")

	viper.SetConfigType("yaml")     // REQUIRED if the config file does not have the extension in the name
	viper.SetConfigName(ConfigName) // name of config file (without extension)

	// bind to environment variables
	openAIEnvs := map[string]string{
		"gpt3.apikey": "GPT3_APIKEY",
		"gpt3.orgid":  "GPT3_ORGID",
		"gpt3.url":    "GPT3_URL",
	}
	for k, v := range openAIEnvs {
		log.Printf("binding %s to %s\n", k, EnvPrefix+"_"+v)
		if err := viper.BindEnv(k, EnvPrefix+"_"+v); err != nil {
			return err
		}
	}
	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if ok := errors.As(err, &configFileNotFound); !ok {
			return err // allow no config file
		}
	}

	// optionally look for local (gitignored) config file and merge it in
	viper.SetConfigName(ConfigFileLocal)

	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if ok := errors.As(err, &configFileNotFound); !ok {
			return err // allow no config file
		}
	}

	bindFlags(cmd, viper.GetViper())
	if err := viper.Unmarshal(c); err != nil {
		return err
	}
	c.PrintAsJSON()
	return nil
}

func (c *Config) PrintAsJSON() {
	vBytes, _ := json.MarshalIndent(c, "", "  ")
	log.Printf("config:\n%s\n", vBytes)
}

// FindFileset Returns a fileset with the matching name,
// or nil if none exists.
func (c *Config) FindFileset(name string) *Filesets {
	for _, fileset := range c.Filesets {
		if fileset.Name == name {
			return &fileset
		}
	}
	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable).
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		log.Println("visiting ", f.Name)
		if f.Changed && !v.IsSet(f.Name) {
			val, _ := cmd.Flags().GetString(f.Name)
			v.Set(f.Name, val)
			log.Printf("setting '%s' to '%s'", f.Name, val)
		}
	})
}
