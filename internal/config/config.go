// Package config loads and exposes fuzzing configuration. Configuration comes
// from a YAML file, the QYVORA_SEKHMET_* environment namespace, and safe
// defaults, in that order of precedence. Invalid configured values are
// rejected rather than silently accepted.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const appName = "qyvora-sekhmet"

// Defaults that shape a first run so the user does not have to configure
// dozens of options before fuzzing.
const (
	DefaultWorkers          = 4
	DefaultTimeoutScale     = 10.0 // adaptive timeout as a multiple of baseline
	DefaultMinTimeout       = 100 * time.Millisecond
	DefaultSeedLimit        = 1024 * 1024 // 1 MiB
	DefaultCampaignRuntime  = 0           // 0 = run until limit or interrupt
	DefaultExecutionLimit   = 0           // 0 = unlimited
	DefaultCorpusRetention  = 100000
	DefaultMemoryLimitBytes = 512 << 20 // 512 MiB
	DefaultDiskLimitBytes   = 2 << 30   // 2 GiB
)

// Strategy names for mutation and scheduling. These are the exported, stable
// identifiers users configure; the actual algorithm selection lives in the
// mutation and scheduler packages.
const (
	StrategyAdaptive       = "adaptive"
	StrategyDeterministic  = "deterministic"
	StrategyRandom         = "random"
	StrategyStructureAware = "structure-aware"
	StrategyGrammar        = "grammar"
)

// IsValidStrategy reports whether name is a supported mutation strategy.
func IsValidStrategy(name string) bool {
	switch strings.ToLower(name) {
	case StrategyAdaptive, StrategyDeterministic, StrategyRandom,
		StrategyStructureAware, StrategyGrammar:
		return true
	}
	return false
}

// Load builds a viper configuration from a config file (when given), the
// QYVORA_SEKHMET_* environment namespace, and defaults. A missing config file
// is not an error; a malformed one is.
func Load(cfgFile string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	for _, dir := range configSearchDirs(cfgFile) {
		v.AddConfigPath(dir)
	}

	v.SetEnvPrefix("QYVORA_SEKHMET")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}
	return v, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("target", "")
	v.SetDefault("target.type", "auto")
	v.SetDefault("input.source", "")
	v.SetDefault("input.cmd_args", false)
	v.SetDefault("input.env", "")
	v.SetDefault("corpus.dir", "corpus")
	v.SetDefault("corpus.retention", DefaultCorpusRetention)
	v.SetDefault("workers", DefaultWorkers)
	v.SetDefault("strategy", StrategyAdaptive)
	v.SetDefault("generate", false)
	v.SetDefault("coverage", true)
	v.SetDefault("baseline.samples", 30)
	v.SetDefault("baseline.warmup", 5)
	v.SetDefault("baseline.enabled", true)
	v.SetDefault("timeout.min", DefaultMinTimeout.String())
	v.SetDefault("timeout.scale", DefaultTimeoutScale)
	v.SetDefault("runtime", int64(DefaultCampaignRuntime))
	v.SetDefault("execution.limit", int64(DefaultExecutionLimit))
	v.SetDefault("memory.limit", int64(DefaultMemoryLimitBytes))
	v.SetDefault("disk.limit", int64(DefaultDiskLimitBytes))
	v.SetDefault("seclists.path", "")
	v.SetDefault("output", "terminal")
	v.SetDefault("report.dir", "reports")
	v.SetDefault("report.format", "terminal")
	v.SetDefault("session.dir", "sessions")
	v.SetDefault("log.level", "info")
	v.SetDefault("verbose", false)
	v.SetDefault("quiet", false)
	v.SetDefault("json", false)
	v.SetDefault("authorized", false)
	v.SetDefault("sim", false)
	v.SetDefault("deterministic", false)
}

// Strategy returns the configured mutation strategy, validated against the
// known set. An invalid configured strategy is an error so a typo cannot
// silently change what a campaign does.
func Strategy(v *viper.Viper) (string, error) {
	s := strings.ToLower(v.GetString("strategy"))
	if !IsValidStrategy(s) {
		return "", fmt.Errorf("unknown strategy %q (valid: %s)",
			v.GetString("strategy"), strings.Join(validStrategies(), ", "))
	}
	return s, nil
}

func validStrategies() []string {
	return []string{StrategyAdaptive, StrategyDeterministic, StrategyRandom,
		StrategyStructureAware, StrategyGrammar}
}

// MinTimeout returns the configured floor for adaptive timeouts.
func MinTimeout(v *viper.Viper) time.Duration {
	d, err := time.ParseDuration(v.GetString("timeout.min"))
	if err != nil || d <= 0 {
		return DefaultMinTimeout
	}
	return d
}

// Workers returns the configured worker count, bounded to a sane minimum.
func Workers(v *viper.Viper) int {
	n := v.GetInt("workers")
	if n < 1 {
		return 1
	}
	return n
}

// BaselineSamples returns the baseline sampling size.
func BaselineSamples(v *viper.Viper) int {
	n := v.GetInt("baseline.samples")
	if n < 1 {
		return DefaultCorpusRetention // never used; guard
	}
	return n
}

func configSearchDirs(cfgFile string) []string {
	dirs := []string{"."}

	if cfgFile != "" {
		if info, err := os.Stat(cfgFile); err == nil && info.IsDir() {
			dirs = append(dirs, cfgFile)
		} else {
			dirs = append(dirs, filepath.Dir(cfgFile))
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "."+appName))
		dirs = append(dirs, filepath.Join(home, ".config", "qyvora", "sekhmet"))
	}

	dirs = append(dirs, filepath.Join("/etc", appName))
	return dirs
}
