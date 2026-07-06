package server

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"hackerwiki/internal/config"
	"hackerwiki/internal/dirwiki"
	"hackerwiki/internal/flatwiki"
	"hackerwiki/internal/gtfobins"
	"hackerwiki/internal/lolbas"
	"hackerwiki/internal/staticsite"
	"hackerwiki/internal/wiki"
	"hackerwiki/internal/xss"
)

//go:embed templates/*.html static/*
var assets embed.FS

type Server struct {
	mux *http.ServeMux
	cfg *config.Config
}

func New(cfg *config.Config) *Server {
	s := &Server{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	wikiTmpl    := mustParse(nil, "wiki.html")
	payloadsTmpl := mustParse(nil, "payloads.html")
	xssTmpl     := mustParse(xss.TemplateFuncs(), "xss.html")

	s.mux.HandleFunc("/", s.landingHandler)
	s.mux.Handle("/static/", http.FileServer(http.FS(assets)))

	if enabled(s.cfg.HackTricksPath) {
		ht := wiki.New("/hacktricks", "HackTricks", s.cfg.HackTricksPath, wikiTmpl)
		s.mux.Handle("/hacktricks/", http.StripPrefix("/hacktricks", ht.Mux()))
		s.mux.Handle("/hacktricks", http.RedirectHandler("/hacktricks/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.RevShellsPath) {
		rs := staticsite.New(s.cfg.RevShellsPath)
		s.mux.Handle("/revshells/", http.StripPrefix("/revshells", rs))
		s.mux.Handle("/revshells", http.RedirectHandler("/revshells/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.XSSCheatsheetPath) {
		dataPath := filepath.Join(s.cfg.XSSCheatsheetPath, "src", "main", "resources", "json")
		xssHandler := xss.New(dataPath, xssTmpl)
		s.mux.Handle("/xss/", http.StripPrefix("/xss", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "" || r.URL.Path == "/" {
				xssHandler.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		})))
		s.mux.Handle("/xss", http.RedirectHandler("/xss/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.CertipyPath) {
		cp := flatwiki.New("/certipy", "Certipy", s.cfg.CertipyPath, wikiTmpl)
		s.mux.Handle("/certipy/", http.StripPrefix("/certipy", cp))
		s.mux.Handle("/certipy", http.RedirectHandler("/certipy/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.PayloadsPath) {
		pw := dirwiki.New("/payloads", "PayloadsAllTheThings", s.cfg.PayloadsPath, payloadsTmpl)
		s.mux.Handle("/payloads/", http.StripPrefix("/payloads", pw.Mux()))
		s.mux.Handle("/payloads", http.RedirectHandler("/payloads/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.CyberChefPath) {
		cc := staticsite.New(s.cfg.CyberChefPath)
		s.mux.Handle("/cyberchef/", http.StripPrefix("/cyberchef", cc))
		s.mux.Handle("/cyberchef", http.RedirectHandler("/cyberchef/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.GTFOBinsPath) {
		gb := gtfobins.New(s.cfg.GTFOBinsPath, mustParse(nil, "gtfobins.html", "gtfobins-bin.html"))
		s.mux.Handle("/gtfobins/", http.StripPrefix("/gtfobins", gb))
		s.mux.Handle("/gtfobins", http.RedirectHandler("/gtfobins/", http.StatusMovedPermanently))
	}

	if enabled(s.cfg.LOLBASPath) {
		lb := lolbas.New(s.cfg.LOLBASPath, mustParse(nil, "lolbas.html", "lolbas-bin.html"))
		s.mux.Handle("/lolbas/", http.StripPrefix("/lolbas", lb))
		s.mux.Handle("/lolbas", http.RedirectHandler("/lolbas/", http.StatusMovedPermanently))
	}
}

func enabled(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func mustParse(funcs template.FuncMap, names ...string) *template.Template {
	base := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}
	for k, v := range funcs {
		base[k] = v
	}
	t := template.New("").Funcs(base)
	patterns := make([]string, 0, len(names)+1)
	patterns = append(patterns, "templates/layout.html")
	for _, n := range names {
		patterns = append(patterns, "templates/"+n)
	}
	return template.Must(t.ParseFS(assets, patterns...))
}
