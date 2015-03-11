package xsdgen

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"regexp"
	"strings"

	"aqwari.net/xml/xsd"
	"golang.org/x/tools/imports"
)

type replaceRule struct {
	from *regexp.Regexp
	to   string
}

type replaceRuleList []replaceRule

func (r *replaceRuleList) String() string {
	var buf bytes.Buffer
	for _, item := range *r {
		fmt.Fprintf(&buf, "%s -> %s\n", item.from, item.to)
	}
	return buf.String()
}

func (r *replaceRuleList) Set(s string) error {
	parts := strings.SplitN(s, "->", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid replace rule %q. must be \"regex -> replacement\"", s)
	}
	parts[0] = strings.TrimSpace(parts[0])
	parts[1] = strings.TrimSpace(parts[1])
	reg, err := regexp.Compile(parts[0])
	if err != nil {
		return fmt.Errorf("invalid regex %q: %v", parts[0], err)
	}
	*r = append(*r, replaceRule{reg, parts[1]})
	return nil
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}

// GenAST creates an *ast.File containing type declarations and
// associated methods based on a set of XML schema.
func (cfg *Config) GenAST(files ...string) (*ast.File, error) {
	data := make([][]byte, 0, len(files))
	for _, filename := range files {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		cfg.debugf("read %s", filename)
		data = append(data, b)
	}
	if len(cfg.namespaces) == 0 {
		cfg.debugf("setting namespaces to %s", cfg.namespaces)
		cfg.Option(Namespaces(lookupTargetNS(data...)...))
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
		return nil, fmt.Errorf("could not find schema for all namespaces in %s",
			cfg.namespaces)
	}

	var file *ast.File
	for _, s := range primaries {
		f, err := cfg.genAST(s, deps...)
		if err != nil {
			return nil, err
		}
		file = mergeASTFile(file, f)
	}

	return file, nil
}

// Generate creates a file containing Go source generated from an XML
// Schema. Main is meant to be called as part of a command, and can
// be used to change the behavior of the xsdgen command in ways that
// its command-line arguments do not allow. The arguments are the
// same as those passed to the xsdgen command.
func (cfg *Config) Generate(arguments ...string) error {
	var (
		err          error
		replaceRules replaceRuleList
		xmlns        stringSlice
		fs           = flag.NewFlagSet("xsdgen", flag.ExitOnError)
		packageName  = fs.String("pkg", "", "name of the the generated package")
		output       = fs.String("o", "xsdgen_output.go", "name of the output file")
	)
	fs.Var(&replaceRules, "r", "replacement rule 'regex -> repl' (can be used multiple times)")
	fs.Var(&xmlns, "ns", "target namespace(s) to generate types for")

	fs.Parse(arguments)
	if fs.NArg() == 0 {
		return errors.New("Usage: xsdgen [-ns xmlns] [-r rule] [-o file] [-pkg pkg] file ...")
	}
	cfg.Option(Namespaces(xmlns...))
	for _, r := range replaceRules {
		cfg.Option(replaceAllNamesRegex(r.from, r.to))
	}
	if *packageName != "" {
		cfg.Option(PackageName(*packageName))
	}

	file, err := cfg.GenAST(fs.Args()...)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	fileset := token.NewFileSet()
	if err := format.Node(&buf, fileset, file); err != nil {
		return err
	}
	out, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(*output, out, 0666)
}
