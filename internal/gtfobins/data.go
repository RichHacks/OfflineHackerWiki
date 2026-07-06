package gtfobins

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultFuncOrder / defaultContextOrder mirror the display order used by the upstream
// _data/functions.yml and _data/contexts.yml files. Any key found in the data files but
// missing from these lists is appended (sorted) so nothing is silently dropped.
var defaultFuncOrder = []string{
	"shell", "command", "reverse-shell", "bind-shell",
	"file-write", "file-read", "upload", "download",
	"library-load", "privilege-escalation", "inherit",
}

var defaultContextOrder = []string{"unprivileged", "sudo", "suid", "capabilities"}

// FuncExtra / ContextExtra hold the free-form "extra" metadata used by a handful of
// function/context kinds to render extra notes (see gtfobin.html upstream layout).
type CodeComment struct {
	Comment string `yaml:"comment"`
	Code    string `yaml:"code"`
}

type FuncExtra struct {
	TTY      map[bool]string        `yaml:"tty"`
	Blind    map[bool]string        `yaml:"blind"`
	Binary   map[bool]string        `yaml:"binary"`
	Listener map[string]CodeComment `yaml:"listener"`
	Connector map[string]CodeComment `yaml:"connector"`
	Receiver map[string]CodeComment `yaml:"receiver"`
	Sender   map[string]CodeComment `yaml:"sender"`
	Payload  string                 `yaml:"payload"`
}

type ContextExtra struct {
	Environment string          `yaml:"environment"`
	Shell       map[bool]string `yaml:"shell"`
	List        string          `yaml:"list"`
}

type FuncMeta struct {
	Key         string
	Label       string    `yaml:"label"`
	Description string    `yaml:"description"`
	Extra       FuncExtra `yaml:"extra"`
}

type ContextMeta struct {
	Key         string
	Label       string       `yaml:"label"`
	Description string       `yaml:"description"`
	Extra       ContextExtra `yaml:"extra"`
}

// Context is a single per-context override under an example's `contexts:` map.
// It may be entirely absent (null) in the YAML, hence pointer fields.
type Context struct {
	Code    string
	Comment string
	Shell   *bool
	List    []string
}

func (c *Context) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		return nil
	}
	var m struct {
		Code    string   `yaml:"code"`
		Comment string   `yaml:"comment"`
		Shell   *bool    `yaml:"shell"`
		List    []string `yaml:"list"`
	}
	if err := value.Decode(&m); err != nil {
		return err
	}
	c.Code, c.Comment, c.Shell, c.List = m.Code, m.Comment, m.Shell, m.List
	return nil
}

// StrOrCC decodes a field that is either a bare string key (referencing a predefined
// entry in the function's "extra" map, e.g. listener/connector/receiver/sender) or an
// inline map with its own comment/code.
type StrOrCC struct {
	Key     string
	Code    string
	Comment string
}

func (s *StrOrCC) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return value.Decode(&s.Key)
	}
	var m CodeComment
	if err := value.Decode(&m); err != nil {
		return err
	}
	s.Code, s.Comment = m.Code, m.Comment
	return nil
}

// Entry is one example ("code") for a given function on a given binary.
// Contexts is nil when the YAML has no `contexts:` key at all (meaning: show this
// example only under the default/first context), as opposed to an explicit (possibly
// empty) map naming which contexts apply.
type Entry struct {
	Code     string              `yaml:"code"`
	Comment  string              `yaml:"comment"`
	Version  string              `yaml:"version"`
	Contexts map[string]*Context `yaml:"contexts"`

	Blind  *bool `yaml:"blind"`
	Binary *bool `yaml:"binary"`
	TTY    *bool `yaml:"tty"`

	From string `yaml:"from"` // used by the "inherit" function

	Listener  *StrOrCC `yaml:"listener"`
	Connector *StrOrCC `yaml:"connector"`
	Receiver  *StrOrCC `yaml:"receiver"`
	Sender    *StrOrCC `yaml:"sender"`
}

// Binary is a single _gtfobins/<name> entry.
type Binary struct {
	Name      string
	Comment   string           `yaml:"comment"`
	Alias     string           `yaml:"alias"`
	Functions map[string][]Entry `yaml:"functions"`
}

// Data is the whole loaded GTFOBins dataset.
type Data struct {
	FuncMeta  map[string]*FuncMeta
	CtxMeta   map[string]*ContextMeta
	FuncOrder []string
	CtxOrder  []string
	Binaries  map[string]*Binary
	Names     []string
}

func Load(dataPath string) (*Data, error) {
	d := &Data{
		FuncMeta: map[string]*FuncMeta{},
		CtxMeta:  map[string]*ContextMeta{},
		Binaries: map[string]*Binary{},
	}

	if raw, err := os.ReadFile(filepath.Join(dataPath, "_data", "functions.yml")); err == nil {
		var m map[string]*FuncMeta
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		for k, v := range m {
			v.Key = k
			d.FuncMeta[k] = v
		}
	}

	if raw, err := os.ReadFile(filepath.Join(dataPath, "_data", "contexts.yml")); err == nil {
		var m map[string]*ContextMeta
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		for k, v := range m {
			v.Key = k
			d.CtxMeta[k] = v
		}
	}

	d.FuncOrder = orderKeys(defaultFuncOrder, keysOfFuncMeta(d.FuncMeta))
	d.CtxOrder = orderKeys(defaultContextOrder, keysOfCtxMeta(d.CtxMeta))

	entries, err := os.ReadDir(filepath.Join(dataPath, "_gtfobins"))
	if err != nil {
		return d, err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dataPath, "_gtfobins", e.Name()))
		if err != nil {
			continue
		}
		var b Binary
		if err := yaml.Unmarshal(raw, &b); err != nil {
			continue
		}
		b.Name = e.Name()
		d.Binaries[b.Name] = &b
	}
	for name := range d.Binaries {
		d.Names = append(d.Names, name)
	}
	sort.Strings(d.Names)

	return d, nil
}

func keysOfFuncMeta(m map[string]*FuncMeta) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func keysOfCtxMeta(m map[string]*ContextMeta) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// orderKeys returns preferred keys (that actually exist in all) followed by any
// remaining keys from all, sorted, so unexpected new data doesn't get dropped.
func orderKeys(preferred, all []string) []string {
	exists := map[string]bool{}
	for _, k := range all {
		exists[k] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, k := range preferred {
		if exists[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	var rest []string
	for _, k := range all {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// Resolve follows a binary's alias chain (if any) to the binary whose Functions
// should actually be displayed.
func (d *Data) Resolve(b *Binary) *Binary {
	seen := map[string]bool{}
	cur := b
	for cur.Alias != "" && !seen[cur.Alias] {
		seen[cur.Alias] = true
		target, ok := d.Binaries[cur.Alias]
		if !ok {
			break
		}
		cur = target
	}
	return cur
}
