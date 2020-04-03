package wsdlgen

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/CognitoIQ/go-xml/internal/commandline"
	"github.com/CognitoIQ/go-xml/internal/gen"
	"github.com/CognitoIQ/go-xml/xsdgen"
)

// The GenSource method converts the AST returned by GenAST to formatted
// Go source code.
func (cfg *Config) GenSource(files ...string) ([]byte, error) {
	file, err := cfg.GenAST(files...)
	if err != nil {
		return nil, err
	}
	return gen.FormattedSource(file, "fixme.go")
}

// GenCLI creates a file containing Go source generated from a WSDL
// definition. It is intended to be called from the main function of any
// command-line interfaces to the wsdlgen package.
func (cfg *Config) GenCLI(arguments ...string) error {
	var (
		err          error
		replaceRules commandline.ReplaceRuleList
		ports        commandline.Strings
		fs           = flag.NewFlagSet("wsdlgen", flag.ExitOnError)
		packageName  = fs.String("pkg", "", "name of the generated package")
		comment      = fs.String("c", "", "First line of package-level comments")
		output       = fs.String("o", "wsdlgen_output.go", "name of the output file")
		verbose      = fs.Bool("v", false, "print verbose output")
		debug        = fs.Bool("vv", false, "print debug output")
	)
	fs.Var(&replaceRules, "r", "replacement rule 'regex -> repl' (can be used multiple times)")
	fs.Var(&ports, "port", "gen code for this port (can be used multiple times)")
	fs.Parse(arguments)
	if fs.NArg() == 0 {
		return errors.New("Usage: wsdlgen [-r rule] [-o file] [-port name] [-pkg pkg] file ...")
	}

	if *debug {
		cfg.Option(LogLevel(5))
	} else if *verbose {
		cfg.Option(LogLevel(1))
	}
	if len(*packageName) > 0 {
		cfg.Option(PackageName(*packageName))
		cfg.XSDOption(xsdgen.PackageName(*packageName))
	}
	if len(*comment) > 0 {
		cfg.Option(PackageComment(*comment))
	}
	if len(ports) > 0 {
		cfg.Option(OnlyPorts(ports...))
	}
	for _, r := range replaceRules {
		cfg.XSDOption(xsdgen.Replace(r.From.String(), r.To))
	}
	file, err := cfg.GenAST(fs.Args()...)
	if err != nil {
		return err
	}

	data, err := gen.FormattedSource(file, *output)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(*output, data, 0666)
}

// The GenCLI function generates Go source code using the default
// options chosen by the wsdlgen package. It is meant to be used from
// the main package of a command-line program.
func GenCLI(args ...string) error {
	var cfg Config
	cfg.Option(DefaultOptions...)
	cfg.XSDOption(xsdgen.DefaultOptions...)
	cfg.Option(LogOutput(log.New(os.Stderr, "", 0)))
	return cfg.GenCLI(args...)
}
