// Usage:
//
//     gopkg [path] [vcs-type] [uri]
//     gopkg [path] [uri]

package gopkg

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("gopkg", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

type Config struct {
	Path      string
	pathRegex *regexp.Regexp
	Vcs       string
	Uri       string
}

type templateVars struct {
	Host string
	Path string
	Vcs  string
	Uri  string
}

type GopkgHandler struct {
	Next    httpserver.Handler
	Configs []Config
}

var tmpl = template.Must(template.New("").Parse(`<html>
<head>
<meta name="go-import" content="{{.Host}}{{.Path}} {{.Vcs}} {{.Uri}}">
<meta name="go-source" content="{{.Host}}{{.Path}} {{.Uri}} {{.Uri}}/tree/master{/dir} {{.Uri}}/blob/master{/dir}/{file}#L{line}" />
</head>
<body>
go get {{.Host}}{{.Path}}
</body>
</html>
`))

func (g GopkgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	vars, err := handleGoPkg(g.Configs, r.Host, r.URL.Path)
	if err != nil {
		return g.Next.ServeHTTP(w, r)
	}

	// Check if the request path contains go-get=1
	if r.FormValue("go-get") != "1" {
		http.Redirect(w, r, vars.Uri, http.StatusTemporaryRedirect)
		return 0, nil
	}

	err = tmpl.Execute(w, vars)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func handleGoPkg(configs []Config, host string, path string) (templateVars, error) {
	for _, cfg := range configs {
		vars, err := cfg.constructTemplateVariables(host, path)
		if err != nil {
			continue
		}

		return vars, nil
	}

	return templateVars{}, errors.New("no matching config")
}

// expandRegex replaces the variables in tmpl with matches from the regex expression in src.
//
// Example: r=asdf/([a-z]+)/asdf, tmpl=$1, src=asdf/this/asdf would return 'this' unquoated.
func expandRegex(r *regexp.Regexp, tmpl string, src string) string {
	var res []byte
	for _, submatches := range r.FindAllStringSubmatchIndex(src, 1) {
		res = r.ExpandString(res, tmpl, src, submatches)
	}
	return string(res)
}

func (c Config) constructTemplateVariables(host string, path string) (templateVars, error) {
	if !c.pathRegex.MatchString(path) {
		return templateVars{}, errors.New("no regex match")
	}

	uri := expandRegex(c.pathRegex, c.Uri, path)
	modPath := expandRegex(c.pathRegex, c.Path, path)

	return templateVars{
		Host: host,
		Path: modPath,
		Vcs:  c.Vcs,
		Uri:  uri,
	}, nil
}

func setup(c *caddy.Controller) error {
	configs, err := parse(c)
	if err != nil {
		return err
	}
	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return GopkgHandler{
			Configs: configs,
			Next:    next,
		}
	})
	return nil
}

var pathReplaceRegex = regexp.MustCompile(`(\$\d)`)

func parse(c *caddy.Controller) ([]Config, error) {
	var configs []Config

	for c.Next() {

		args := c.RemainingArgs()

		if len(args) != 2 && len(args) != 3 {
			return configs, c.ArgErr()
		}

		cfg := Config{
			Vcs:  "git",
			Path: args[0],
		}

		if len(args) == 2 {
			cfg.Uri = args[1]
		} else {
			cfg.Vcs = args[1]
			cfg.Uri = args[2]
		}

		path := pathReplaceRegex.ReplaceAllString(cfg.Path, `([\w-_]+)`)
		pathRegex, err := regexp.Compile(path)
		if err != nil {
			return configs, err
		}
		cfg.pathRegex = pathRegex

		configs = append(configs, cfg)
	}

	return configs, nil
}
