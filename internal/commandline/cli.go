// Package commandline contains helper types for collecting
// command-line arguments.
package commandline // import "aqwari.net/xml/internal/commandline"

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// A ReplaceRule maps a pattern to its replacement. On the
// command line, ReplaceRules are provided as strings separated
// by "->".
type ReplaceRule struct {
	From *regexp.Regexp
	To   string
}

// A ReplaceRuleList is used to collect multiple replacement rules
// from the command line.
type ReplaceRuleList []ReplaceRule

func (r *ReplaceRuleList) String() string {
	var buf bytes.Buffer
	for _, item := range *r {
		fmt.Fprintf(&buf, "%s -> %s\n", item.From, item.To)
	}
	return buf.String()
}

// Set adds a replacement rule to the ReplaceRuleList, in the order
// provided on the command line.
func (r *ReplaceRuleList) Set(s string) error {
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
	*r = append(*r, ReplaceRule{reg, parts[1]})
	return nil
}

// The Strings type can be used to collect multiple command-line options,
// in the order provided.
type Strings []string

func (s *Strings) String() string {
	return strings.Join(*s, ",")
}

func (s *Strings) Set(val string) error {
	*s = append(*s, val)
	return nil
}
