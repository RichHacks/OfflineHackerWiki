package lolbas

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// categoryOrder mirrors the yml/ subdirectory layout of the upstream LOLBAS repo,
// which doubles as the "Type" facet on the real site.
var categoryOrder = []string{"OSBinaries", "OSLibraries", "OSScripts", "OtherMSBinaries", "HonorableMentions"}

// FuncMeta describes a command "Category" (Execute, Download, AWL Bypass, ...).
// This is small, curated, and stable upstream (site _data/functions.yml), so it's
// embedded here rather than fetched from a second repo.
type FuncMeta struct {
	Key         string
	Label       string
	Description string
}

var functionMeta = []FuncMeta{
	{"execute", "Execute", "Can achieve arbitrary code execution."},
	{"download", "Download", "Can download files."},
	{"upload", "Upload", "Can upload files."},
	{"encode", "Encode", "Can encode files."},
	{"decode", "Decode", "Can decode files."},
	{"ads", "Alternate data streams", "Can write or read alternate data streams."},
	{"copy", "Copy", "Can copy a file."},
	{"credentials", "Credentials", "Can view a credentials file."},
	{"compile", "Compile", "Can compile code."},
	{"awl bypass", "AWL bypass", "Can bypass application allowlisting solutions."},
	{"uac bypass", "UAC bypass", "Can bypass User Access Control in Windows."},
	{"reconnaissance", "Reconnaissance", "Can be used to gather interesting information."},
	{"dump", "Dump", "Can be used to dump the memory contents of a process."},
	{"tamper", "Tamper", "Can be used to tamper with files, processes, etc."},
	{"conceal", "Conceal", "Can be used to conceal malicious activity."},
}

var functionMetaByKey = func() map[string]FuncMeta {
	m := make(map[string]FuncMeta, len(functionMeta))
	for _, f := range functionMeta {
		m[f.Key] = f
	}
	return m
}()

type AliasEntry struct {
	Alias string `yaml:"Alias"`
}
type PathEntry struct {
	Path string `yaml:"Path"`
}
type CodeEntry struct {
	Code string `yaml:"Code"`
}
type ResourceEntry struct {
	Link string `yaml:"Link"`
}
type Acknowledgement struct {
	Person string `yaml:"Person"`
	Handle string `yaml:"Handle"`
}

type Command struct {
	Command         string              `yaml:"Command"`
	Description     string              `yaml:"Description"`
	Usecase         string              `yaml:"Usecase"`
	Category        string              `yaml:"Category"`
	Privileges      string              `yaml:"Privileges"`
	MitreID         string              `yaml:"MitreID"`
	OperatingSystem string              `yaml:"OperatingSystem"`
	Tags            []map[string]string `yaml:"Tags"`
}

// CategoryKey normalizes Category ("AWL Bypass") to the functionMeta key ("awl bypass").
func (c Command) CategoryKey() string {
	return strings.ToLower(strings.TrimSpace(c.Category))
}

type Binary struct {
	Name            string              `yaml:"Name"`
	Description     string              `yaml:"Description"`
	Aliases         []AliasEntry        `yaml:"Aliases"`
	Author          string              `yaml:"Author"`
	Created         string              `yaml:"Created"`
	Commands        []Command           `yaml:"Commands"`
	FullPath        []PathEntry         `yaml:"Full_Path"`
	CodeSample      []CodeEntry         `yaml:"Code_Sample"`
	Detection       []map[string]string `yaml:"Detection"`
	Resources       []ResourceEntry     `yaml:"Resources"`
	Acknowledgement []Acknowledgement   `yaml:"Acknowledgement"`

	Category string `yaml:"-"` // OSBinaries / OSLibraries / OSScripts / OtherMSBinaries / HonorableMentions
	Slug     string `yaml:"-"` // filename stem, unique within Category
}

// URLPath is how this binary is addressed under the /lolbas/ mount.
func (b *Binary) URLPath() string {
	return b.Category + "/" + b.Slug
}

// Functions returns the distinct Category keys used by this binary's commands, in
// functionMeta display order.
func (b *Binary) Functions() []string {
	set := map[string]bool{}
	for _, c := range b.Commands {
		set[c.CategoryKey()] = true
	}
	var out []string
	for _, f := range functionMeta {
		if set[f.Key] {
			out = append(out, f.Key)
		}
	}
	return out
}

// MitreIDs returns the distinct, non-empty MITRE ATT&CK technique IDs referenced by
// this binary's commands.
func (b *Binary) MitreIDs() []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range b.Commands {
		id := strings.TrimSpace(c.MitreID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// CommandsByCategory groups this binary's commands by Category, in functionMeta order.
func (b *Binary) CommandsByCategory() []struct {
	Key      string
	Commands []Command
} {
	byKey := map[string][]Command{}
	for _, c := range b.Commands {
		k := c.CategoryKey()
		byKey[k] = append(byKey[k], c)
	}
	var out []struct {
		Key      string
		Commands []Command
	}
	for _, f := range functionMeta {
		if cmds, ok := byKey[f.Key]; ok {
			out = append(out, struct {
				Key      string
				Commands []Command
			}{f.Key, cmds})
		}
	}
	return out
}

type Data struct {
	Binaries map[string]*Binary // by Category+"/"+Slug
	Names    []string           // sorted keys into Binaries, for stable index order
}

func Load(dataPath string) (*Data, error) {
	d := &Data{Binaries: map[string]*Binary{}}

	for _, cat := range categoryOrder {
		dir := filepath.Join(dataPath, "yml", cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".yml") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var b Binary
			if err := yaml.Unmarshal(raw, &b); err != nil {
				continue
			}
			b.Category = cat
			b.Slug = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			key := b.URLPath()
			d.Binaries[key] = &b
		}
	}

	for k := range d.Binaries {
		d.Names = append(d.Names, k)
	}
	sort.Slice(d.Names, func(i, j int) bool {
		return strings.ToLower(d.Binaries[d.Names[i]].Name) < strings.ToLower(d.Binaries[d.Names[j]].Name)
	})

	return d, nil
}
