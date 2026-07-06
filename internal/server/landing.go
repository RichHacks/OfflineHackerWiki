package server

import (
	"html/template"
	"log"
	"net/http"
)

type tool struct {
	ID          string
	Name        string
	Description string
	URL         string
	Enabled     bool
	Icon        string
	Tag         string
}

type landingData struct {
	Tools []tool
}

func (s *Server) landingHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tools := []tool{
		{
			ID:          "hacktricks",
			Name:        "HackTricks",
			Description: "The comprehensive hacking reference: pentesting methodology, exploits, privilege escalation, and more.",
			URL:         "/hacktricks/",
			Enabled:     enabled(s.cfg.HackTricksPath),
			Icon:        "🃏",
			Tag:         "Wiki",
		},
		{
			ID:          "payloads",
			Name:        "PayloadsAllTheThings",
			Description: "A curated list of useful payloads and bypasses for web application security.",
			URL:         "/payloads/",
			Enabled:     enabled(s.cfg.PayloadsPath),
			Icon:        "💣",
			Tag:         "Payloads",
		},
		{
			ID:          "revshells",
			Name:        "Reverse Shell Generator",
			Description: "Generate reverse shell one-liners for Bash, Python, Perl, PowerShell, Netcat, and many more.",
			URL:         "/revshells/",
			Enabled:     enabled(s.cfg.RevShellsPath),
			Icon:        "🐚",
			Tag:         "Tool",
		},
		{
			ID:          "xss",
			Name:        "XSS Cheat Sheet",
			Description: "PortSwigger's cross-site scripting cheat sheet: vectors, events, browsers, and bypass techniques.",
			URL:         "/xss/",
			Enabled:     enabled(s.cfg.XSSCheatsheetPath),
			Icon:        "⚡",
			Tag:         "Cheat Sheet",
		},
		{
			ID:          "certipy",
			Name:        "Certipy Wiki",
			Description: "Active Directory Certificate Services enumeration and exploitation with Certipy.",
			URL:         "/certipy/",
			Enabled:     enabled(s.cfg.CertipyPath),
			Icon:        "📜",
			Tag:         "AD / PKI",
		},
		{
			ID:          "cyberchef",
			Name:        "CyberChef",
			Description: "The Cyber Swiss Army Knife: a web app for encryption, encoding, compression, and data analysis.",
			URL:         "/cyberchef/",
			Enabled:     enabled(s.cfg.CyberChefPath),
			Icon:        "🧪",
			Tag:         "Tool",
		},
		{
			ID:          "gtfobins",
			Name:        "GTFOBins",
			Description: "Unix executables that can be abused to bypass local security restrictions in misconfigured systems.",
			URL:         "/gtfobins/",
			Enabled:     enabled(s.cfg.GTFOBinsPath),
			Icon:        "🐧",
			Tag:         "Cheat Sheet",
		},
		{
			ID:          "lolbas",
			Name:        "LOLBAS",
			Description: "Living Off The Land Binaries, Scripts and Libraries: Windows executables abused for offensive purposes.",
			URL:         "/lolbas/",
			Enabled:     enabled(s.cfg.LOLBASPath),
			Icon:        "🪟",
			Tag:         "Cheat Sheet",
		},
	}

	tmpl := template.Must(template.New("landing").ParseFS(assets, "templates/landing.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "landing.html", landingData{Tools: tools}); err != nil {
		log.Printf("landing template error: %v", err)
	}
}

