package wsdlgen

import (
	"errors"
	"flag"
	"io/ioutil"

	"aqwari.net/xml/internal/commandline"
	"aqwari.net/xml/internal/gen"
)

// The GenSource method converts the AST returned by GenAST to formatted
// Go source code.
func (cfg *Config) GenSource(files ...string) ([]byte, error) {
	file, err := cfg.GenAST(files...)
	if err != nil {
		return nil, err
	}
	return gen.FormattedSource(file)
}

// GenCLI creates a file containing Go source generated from a WSDL
// definition. It is intended to be called from the main function of any
// command-line interfaces to the wsdlgen package.
func (cfg *Config) GenCLI(arguments ...string) error {
	var (
		err          error
		replaceRules commandline.ReplaceRuleList
		fs           = flag.NewFlagSet("wsdlgen", flag.ExitOnError)
		packageName  = fs.String("pkg", "", "name of the generated package")
		output       = fs.String("o", "wsdlgen_output.go", "name of the output file")
		verbose      = fs.Bool("v", false, "print verbose output")
		debug        = fs.Bool("vv", false, "print debug output")
	)
	fs.Var(&replaceRules, "r", "replacement rule 'regex -> repl' (can be used multiple times)")
	fs.Parse(arguments)
	if fs.NArg() == 0 {
		return errors.New("Usage: wsdlgen [-r rule] [-o file] [-pkg pkg] file ...")
	}

	if *debug {
		cfg.Option(LogLevel(5))
	} else if *verbose {
		cfg.Option(LogLevel(1))
	}
	if len(*packageName) > 0 {
		cfg.Option(PackageName(*packageName))
	}

	file, err := cfg.GenAST(fs.Args()...)
	if err != nil {
		return err
	}

	data, err := gen.FormattedSource(file)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(*output, data, 0666)
}
