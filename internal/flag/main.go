package flag

import (
	"flag"
	"strings"
)

func Parse() {
	flag.Parse()
}

type stringSlice []string

func (s stringSlice) String() string {
	return strings.Join(s, ", ")
}
func (s *stringSlice) Set(value string) error {
	*s = stringSlice(append([]string(*s), value))
	return nil
}
func (s stringSlice) Get() any { return []string(s) }

func StringVar(p *string, name, value, usage string) {
	flag.StringVar(p, name, value, usage)
}
func IntVar(p *int, name string, value int, usage string) {
	flag.IntVar(p, name, value, usage)
}

func StringSliceVar(slice *[]string, name, usage string) {
	flag.Var((*stringSlice)(slice), name, usage)
}
