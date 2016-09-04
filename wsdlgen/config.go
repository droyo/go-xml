package wsdlgen

import (
	"encoding/xml"

	"aqwari.net/xml/wsdl"
	"aqwari.net/xml/xsdgen"
)

// A Config contains parameters for the various code generation processes.
// Users may modify the output of the wsdlgen package's code generation
// by using a Config's Option method to change these parameters.
type Config struct {
	pkgName    string
	pkgHeader  string
	logger     Logger
	loglevel   int
	xsdgen     xsdgen.Config
	portFilter func(wsdl.Port) bool

	maxArgs, maxReturns int
}

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

// The OnlyPorts option defines a whitelist of WSDL ports to generate
// code for. Any other ports will not have types or methods present in
// the generated output.
func OnlyPorts(ports ...string) Option {
	return func(cfg *Config) Option {
		cfg.portFilter = func(p wsdl.Port) bool {
			for _, name := range ports {
				if name == p.Name {
					return true
				}
			}
			return false
		}
		return OnlyPorts()
	}
}

// PackageName specifies the name of the generated Go package.
func PackageName(name string) Option {
	return func(cfg *Config) Option {
		prev := cfg.pkgName
		cfg.pkgName = name
		return PackageName(prev)
	}
}

// PackageComment specifies the first line of package-level Godoc comments.
// If the input WSDL file provides package-level comments, they are added after
// the provided comment, separated by a newline.
func PackageComment(comment string) Option {
	return func(cfg *Config) Option {
		prev := cfg.pkgHeader
		cfg.pkgHeader = comment
		return PackageComment(prev)
	}
}

// LogLevel sets the level of verbosity for log messages generated during
// the code generation process.
func LogLevel(level int) Option {
	return func(cfg *Config) Option {
		prev := cfg.loglevel
		cfg.loglevel = level
		cfg.xsdgen.Option(xsdgen.LogLevel(level))
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

// InputThreshold sets the maximum number of parameters a
// generated function may take. If a WSDL operation is defined as
// taking greater than n parameters, the generated function will
// take only one parameter; a struct, through which all arguments
// will be accessed.
func InputThreshold(n int) Option {
	return func(cfg *Config) Option {
		prev := cfg.maxArgs
		cfg.maxArgs = n
		return InputThreshold(prev)
	}
}

// OutputThreshold sets the maximum number of values that a
// generated function may return. If a WSDL operation is defined
// as returning greater than n values, the generated function will
// return a wrapper struct instead. Note that the error value that all
// generated functions return is not counted against the threshold.
func OutputThreshold(n int) Option {
	return func(cfg *Config) Option {
		prev := cfg.maxReturns
		cfg.maxReturns = n
		return OutputThreshold(prev)
	}
}
