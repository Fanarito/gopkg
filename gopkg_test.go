package gopkg

import (
	"testing"

	"github.com/caddyserver/caddy"
)

func TestGopkgConfig(t *testing.T) {
	type io struct {
		host    string
		path    string
		wantErr bool
		o       templateVars
	}
	tests := []struct {
		input     string
		shouldErr bool
		expect    []Config
		io        []io
	}{
		// Single config
		{
			`gopkg /chrisify https://github.com/zikes/chrisify`,
			false,
			[]Config{
				{
					Path: "/chrisify",
					Vcs:  "git",
					Uri:  "https://github.com/zikes/chrisify",
				},
			},
			[]io{},
		},
		// Multiple config
		{
			`
			gopkg /chrisify https://github.com/zikes/chrisify
			gopkg /multistatus https://github.com/zikes/multistatus
			`,
			false,
			[]Config{
				{
					Path: "/chrisify",
					Vcs:  "git",
					Uri:  "https://github.com/zikes/chrisify",
				},
				{
					Path: "/multistatus",
					Vcs:  "git",
					Uri:  "https://github.com/zikes/multistatus",
				},
			},
			[]io{},
		},
		// Mercurial
		{
			`gopkg /myrepo hg https://bitbucket.org/zikes/myrepo`,
			false,
			[]Config{
				{
					Path: "/myrepo",
					Vcs:  "hg",
					Uri:  "https://bitbucket.org/zikes/myrepo",
				},
			},
			[]io{},
		},
		// Regex
		{
			`gopkg /github/$1/$2 https://github.com/$1/$2`,
			false,
			[]Config{
				{
					Path: `/github/$1/$2`,
					Vcs:  "git",
					Uri:  `https://github.com/$1/$2`,
				},
			},
			[]io{
				{
					"example.com",
					"/github/xxx/yyy",
					false,
					templateVars{
						Host: "example.com",
						Path: "/github/xxx/yyy",
						Vcs:  "git",
						Uri:  "https://github.com/xxx/yyy",
					},
				},
			},
		},
		// GitLab subgroup + no subgroup regex
		{
			`
			gopkg /$1/$2 https://gitlab.com/exampleorg/$1/$2
			gopkg /$1 https://gitlab.com/exampleorg/$1
			`,
			false,
			[]Config{
				{
					Path: `/$1/$2`,
					Vcs:  "git",
					Uri:  `https://gitlab.com/exampleorg/$1/$2`,
				},
				{
					Path: `/$1`,
					Vcs:  "git",
					Uri:  `https://gitlab.com/exampleorg/$1`,
				},
			},
			[]io{
				{
					"example.com",
					"/backend/api",
					false,
					templateVars{
						Host: "example.com",
						Path: "/backend/api",
						Vcs:  "git",
						Uri:  "https://gitlab.com/exampleorg/backend/api",
					},
				},
				{
					"example.com",
					"/api",
					false,
					templateVars{
						Host: "example.com",
						Path: "/api",
						Vcs:  "git",
						Uri:  "https://gitlab.com/exampleorg/api",
					},
				},
			},
		},
		// Subpackages get the modules url
		{
			`gopkg /github/$1/$2 https://github.com/$1/$2`,
			false,
			[]Config{
				{
					Path: `/github/$1/$2`,
					Vcs:  "git",
					Uri:  `https://github.com/$1/$2`,
				},
			},
			[]io{
				{
					"example.com",
					"/github/xxx/yyy/zzz",
					false,
					templateVars{
						Host: "example.com",
						Path: "/github/xxx/yyy",
						Vcs:  "git",
						Uri:  "https://github.com/xxx/yyy",
					},
				},
			},
		},
	}

	for _, test := range tests {
		c := caddy.NewTestController("http", test.input)
		actual, err := parse(c)
		if !test.shouldErr && err != nil {
			t.Errorf("Unexpected error with %v:\n  %v\n", test.input, err)
		}
		if test.shouldErr && err == nil {
			t.Errorf("Expected error with %v but got none\n", test.input)
		}

		for idx, cfg := range test.expect {
			actualCfg := actual[idx]
			if cfg.Path != actualCfg.Path {
				t.Errorf(
					"Mismatched Path config in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					cfg.Path,
					actualCfg.Path,
				)
			}
			if cfg.Vcs != actualCfg.Vcs {
				t.Errorf(
					"Mismatched Vcs config in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					cfg.Vcs,
					actualCfg.Vcs,
				)
			}
			if cfg.Uri != actualCfg.Uri {
				t.Errorf(
					"Mismatched Uri config in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					cfg.Uri,
					actualCfg.Uri,
				)
			}
		}

		for _, io := range test.io {
			vars, err := handleGoPkg(actual, io.host, io.path)
			if err != nil && !io.wantErr {
				t.Errorf("Got error when no error was expected: %v", err)
			} else if err == nil && io.wantErr {
				t.Errorf("Received no error when an error was expected")
			}

			if vars.Host != io.o.Host {
				t.Errorf(
					"Mismatched Host variable in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					io.o.Host,
					vars.Host,
				)
			}

			if vars.Vcs != io.o.Vcs {
				t.Errorf(
					"Mismatched Vcs variable in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					io.o.Vcs,
					vars.Vcs,
				)
			}

			if vars.Path != io.o.Path {
				t.Errorf(
					"Mismatched Path variable in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					io.o.Path,
					vars.Path,
				)
			}

			if vars.Uri != io.o.Uri {
				t.Errorf(
					"Mismatched Uri variable in %v, expected\n  %v\ngot\n  %v\n",
					test.input,
					io.o.Uri,
					vars.Uri,
				)
			}
		}
	}
}
