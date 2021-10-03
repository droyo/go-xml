package xsdgen

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"io/ioutil"
	"path/filepath"
	"strings"

	"aqwari.net/xml/internal/commandline"
	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xsd"
)

// GenCode reads all xml schema definitions from the provided
// data. If succesful, the returned *Code value can be used to
// lookup identifiers and generate Go code.
func (cfg *Config) GenCode(data ...[]byte) (*Code, error) {
	if len(cfg.namespaces) == 0 {
		cfg.Option(Namespaces(lookupTargetNS(data...)...))
		cfg.debugf("setting namespaces to %q", cfg.namespaces)
	}
	deps, err := xsd.Parse(data...)
	if err != nil {
		return nil, err
	}
	primaries := make([]xsd.Schema, 0, len(cfg.namespaces))
	for _, s := range deps {
		for _, ns := range cfg.namespaces {
			if s.TargetNS == ns {
				primaries = append(primaries, s)
				break
			}
		}
	}
	if len(primaries) < len(cfg.namespaces) {
		missing := make([]string, 0, len(cfg.namespaces)-len(primaries))
		have := make(map[string]bool)
		for _, schema := range primaries {
			have[schema.TargetNS] = true
		}
		for _, ns := range cfg.namespaces {
			if !have[ns] {
				missing = append(missing, ns)
			}
		}
		return nil, fmt.Errorf("could not find schema for %q", strings.Join(missing, ", "))
	}
	cfg.addStandardHelpers()
	return cfg.gen(primaries, deps)
}

// GenAST creates an *ast.File containing type declarations and
// associated methods based on a set of XML schema.
func (cfg *Config) GenAST(files ...string) (*ast.File, error) {
	data, err := cfg.readFiles(files...)
	code, err := cfg.GenCode(data...)
	if err != nil {
		return nil, err
	}
	return code.GenAST()
}

func (cfg *Config) readFiles(files ...string) ([][]byte, error) {
	data := make([][]byte, 0, len(files))
	for _, filename := range files {
		path, err := filepath.Abs(filename)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		cfg.debugf("read %s(%s)", path, filename)
		if cfg.followImports {
			dir := filepath.Dir(path)
			importedRefs, err := xsd.Imports(b)
			if err != nil {
				return nil, fmt.Errorf("error discovering imports: %v", err)
			}
			importedFiles := make([]string, 0, len(importedRefs))
			for _, r := range importedRefs {
				if filepath.IsAbs(r.Location) {
					importedFiles = append(importedFiles, r.Location)
				} else {
					importedFiles = append(importedFiles, filepath.Join(dir, r.Location))
				}
			}
			referencedData, err := cfg.readFiles(importedFiles...)
			if err != nil {
				return nil, fmt.Errorf("error reading imported files: %v", err)
			}
			for _, d := range referencedData {
				// prepend imported refs (i.e. append before the referencing file)
				data = append(data, d)
			}
		}
		data = append(data, b)
	}
	return data, nil
}

// The GenSource method converts the AST returned by GenAST to formatted
// Go source code.
func (cfg *Config) GenSource(files ...string) ([]byte, error) {
	file, err := cfg.GenAST(files...)
	if err != nil {
		return nil, err
	}
	return gen.FormattedSource(file, "fixme.go")
}

// GenCLI creates a file containing Go source generated from an XML
// Schema. Main is meant to be called as part of a command, and can
// be used to change the behavior of the xsdgen command in ways that
// its command-line arguments do not allow. The arguments are the
// same as those passed to the xsdgen command.
func (cfg *Config) GenCLI(arguments ...string) error {
	var (
		err           error
		replaceRules  commandline.ReplaceRuleList
		xmlns         commandline.Strings
		fs            = flag.NewFlagSet("xsdgen", flag.ExitOnError)
		packageName   = fs.String("pkg", "", "name of the the generated package")
		output        = fs.String("o", "xsdgen_output.go", "name of the output file")
		followImports = fs.Bool("f", false, "follow import statements; load imported references recursively into scope")
		verbose       = fs.Bool("v", false, "print verbose output")
		debug         = fs.Bool("vv", false, "print debug output")
	)
	fs.Var(&replaceRules, "r", "replacement rule 'regex -> repl' (can be used multiple times)")
	fs.Var(&xmlns, "ns", "target namespace(s) to generate types for")

	if err = fs.Parse(arguments); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("Usage: xsdgen [-ns xmlns] [-r rule] [-o file] [-pkg pkg] file ...")
	}
	if *debug {
		cfg.Option(LogLevel(5))
	} else if *verbose {
		cfg.Option(LogLevel(1))
	}
	cfg.Option(Namespaces(xmlns...))
	cfg.Option(FollowImports(*followImports))
	for _, r := range replaceRules {
		cfg.Option(replaceAllNamesRegex(r.From, r.To))
	}
	if *packageName != "" {
		cfg.Option(PackageName(*packageName))
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
