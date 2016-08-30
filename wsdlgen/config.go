package wsdlgen

import (
	"encoding/xml"

	"aqwari.net/xml/xsdgen"
)

func init() {
	defaultConfig.Option(DefaultOptions...)
}

// A Config contains parameters for the various code generation processes.
// Users may modify the output of the wsdlgen package's code generation
// by using a Config's Option method to change these parameters.
type Config struct {
	pkgName   string
	pkgHeader string
	logger    Logger
	loglevel  int
	xsdgen    xsdgen.Config
}

var defaultConfig Config

func (cfg *Config) logf(format string, args ...interface{}) {
	if cfg.logger != nil {
		cfg.logger.Printf(format, args...)
	}
}

func (cfg *Config) verbosef(format string, args ...interface{}) {
	if cfg.loglevel > 0 {
		cfg.logf(format, args...)
	}
}

func (cfg *Config) debugf(format string, args ...interface{}) {
	if cfg.loglevel > 2 {
		cfg.logf(format, args...)
	}
}

func (cfg *Config) publicName(name xml.Name) string {
	return cfg.xsdgen.NameOf(name)
}

// Option applies the provides Options to a Config, modifying the
// code generation process. The return value of Option can be
// used to revert the effects of the final parameter.
func (cfg *Config) Option(opts ...Option) (previous Option) {
	for _, opt := range opts {
		previous = opt(cfg)
	}
	return previous
}

// XSDOption controls the generation of type declarations according
// to the xsdgen package.
func (cfg *Config) XSDOption(opts ...xsdgen.Option) (previous xsdgen.Option) {
	return cfg.xsdgen.Option(opts...)
}

// An Option modifies code generation parameters. The return value of an
// Option can be used to undo its effect.
type Option func(*Config) Option

// DefaultOptions are the default options for Go source code generation.
var DefaultOptions = []Option{
	PackageName("ws"),
}

// PackageName specifies the name of the generated Go package.
func PackageName(name string) Option {
	return func(cfg *Config) Option {
		prev := cfg.pkgName
		cfg.pkgName = name
		return PackageName(prev)
	}
}

// LogLevel sets the level of verbosity for log messages generated during
// the code generation process.
func LogLevel(level int) Option {
	return func(cfg *Config) Option {
		prev := cfg.loglevel
		cfg.loglevel = level
		return LogLevel(prev)
	}
}

// LogOutput sets the destination for log messages generated during
// code generation.
func LogOutput(dest Logger) Option {
	return func(cfg *Config) Option {
		prev := cfg.logger
		cfg.logger = dest
		cfg.xsdgen.Option(xsdgen.LogOutput(dest))
		return LogOutput(prev)
	}
}
