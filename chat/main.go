package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/Asuforce/gogo/trace"
	"github.com/BurntSushi/toml"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
)

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

type tomlConfig struct {
	SecurityKey string `toml:"security_key"`
	Facebook    authInfo
	GitHub      authInfo
	Google      authInfo
}

type authInfo struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(
			template.ParseFiles(
				filepath.Join("templates", t.filename),
			),
		)
	})
	if err := t.templ.Execute(w, r); err != nil {
		log.Fatal("TemplateExecution:", err)
	}
}

func main() {
	var addr = flag.String("addr", ":8080", "Application address")
	flag.Parse()

	var c tomlConfig
	if _, err := toml.DecodeFile("./config.local.toml", &c); err != nil {
		log.Fatalf("Failed to decode file : %s(MISSING)", err)
	}

	gomniauth.SetSecurityKey(c.SecurityKey)
	gomniauth.WithProviders(
		facebook.New(c.Facebook.ClientID, c.Facebook.ClientSecret, "http://localhost:8080/auth/callback/facebook"),
		github.New(c.GitHub.ClientID, c.GitHub.ClientSecret, "http://localhost:8080/auth/callback/github"),
		google.New(c.Google.ClientID, c.Google.ClientSecret, "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	r.tracer = trace.New(os.Stdout)

	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)

	go r.run()

	log.Println("Start web server. port:", *addr)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
