package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/docker/distribution/reference"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/sshforward/sshprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Build struct {
	AddHost   []string `usage:"Add a custom host-to-IP mapping (host:ip)"`
	BuildArg  []string `usage:"Set build-time variables"`
	CacheFrom []string `usage:"Images to consider as cache sources"`
	File      string   `usage:"Name of the Dockerfile (Default is 'PATH/Dockerfile')" short:"f"`
	Label     []string `usage:"Set metadata for an image"`
	NoCache   bool     `usage:"Do not use cache when building the image"`
	Output    []string `usage:"BuildKit-style output directives (e.g. type=local,dest=path/to/output-dir)" short:"o" slice:"array"`
	Progress  string   `usage:"Set type of progress output (auto, plain, tty). Use plain to show container output" default:"auto"`
	Quiet     bool     `usage:"Suppress the build output and print image ID on success" short:"q"`
	Tag       []string `usage:"Name and optionally a tag in the 'name:tag' format" short:"t"`
	Target    string   `usage:"Set the target build stage to build."`
	Pull      bool     `usage:"Always attempt to pull a newer version of the image"`
	Secret    []string `usage:"Secret value exposed to the build. Format id=secretname|src=filepath" slice:"array"`
	Ssh       []string `usage:"Allow forwarding SSH agent to the builder. Format default|<id>[=<socket>|<key>[,<key>]]" slice:"array"`
}

func (s *Build) Do(ctx context.Context, k8s *client.Interface, path string) error {
	return client.Control(ctx, k8s, func(ctx context.Context, bkc *buildkit.Client) error {
		options := buildkit.SolveOpt{
			Frontend:      "dockerfile.v0",
			FrontendAttrs: s.frontendAttrs(),
			CacheImports:  s.cacheImports(),
			LocalDirs:     s.localDirs(path),
			Session:       []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
		}
		if len(s.Tag) > 0 {
			options.Exports = s.defaultExporter()
		}
		if len(s.Output) > 0 {
			exports, err := build.ParseOutput(s.Output)
			if err != nil {
				return err
			}
			options.Exports = append(options.Exports, exports...)
		}
		if len(s.Secret) > 0 {
			attachable, err := build.ParseSecret(s.Secret)
			if err != nil {
				return err
			}
			options.Session = append(options.Session, attachable)
		}
		if len(s.Ssh) > 0 {
			configs, err := build.ParseSSH(s.Ssh)
			if err != nil {
				return err
			}
			attachable, err := sshprovider.NewSSHAgentProvider(configs)
			if err != nil {
				return err
			}
			options.Session = append(options.Session, attachable)
		}
		if s.Quiet {
			s.Progress = "none"
		}
		eg := errgroup.Group{}
		res, err := bkc.Solve(ctx, nil, options, s.progress(&eg))
		if err != nil {
			return err
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		logrus.Debugf("%#v", res)
		if s.Quiet && res.ExporterResponse != nil {
			if id := res.ExporterResponse["containerimage.config.digest"]; id != "" {
				fmt.Fprintln(os.Stdout, id)
			}
		}
		return nil
	})
}

func (s *Build) frontendAttrs() map[string]string {
	// --target
	m := map[string]string{
		"target": s.Target,
	}
	// --build-arg
	for _, b := range s.BuildArg {
		p := strings.SplitN(b, "=", 2)
		k := fmt.Sprintf("build-arg:%s", p[0])
		v := strings.Join(p[1:], "=")
		m[k] = v
	}
	// --label
	for _, l := range s.Label {
		p := strings.SplitN(l, "=", 2)
		k := fmt.Sprintf("label:%s", p[0])
		v := strings.Join(p[1:], "=")
		m[k] = v
	}
	// --add-host
	h := strings.Join(s.AddHost, ",")
	if h != "" {
		m["add-hosts"] = h
	}
	// --file
	if s.File == "" {
		m["filename"] = "Dockerfile"
	} else {
		m["filename"] = filepath.Base(s.File)
	}
	// --no-cache
	if s.NoCache {
		m["no-cache"] = "" // true
	}
	// --pull
	if s.Pull {
		m["image-resolve-mode"] = "pull"
	}
	return m
}

func (s *Build) localDirs(path string) map[string]string {
	m := map[string]string{
		"context": path,
	}
	if s.File == "" {
		m["dockerfile"] = path
	} else {
		m["dockerfile"] = filepath.Dir(s.File)
	}
	return m
}

func (s *Build) cacheImports() (result []buildkit.CacheOptionsEntry) {
	exists := map[string]bool{}
	for _, s := range s.CacheFrom {
		if exists[s] {
			continue
		}
		exists[s] = true

		result = append(result, buildkit.CacheOptionsEntry{
			Type: "registry",
			Attrs: map[string]string{
				"ref": s,
			},
		})
	}

	return
}

func (s *Build) progress(group *errgroup.Group) chan *buildkit.SolveStatus {
	var (
		c   console.Console
		err error
	)

	switch s.Progress {
	case "none":
		return nil
	case "plain":
	default:
		c, err = console.ConsoleFromFile(os.Stderr)
		if err != nil {
			c = nil
		}
	}

	ch := make(chan *buildkit.SolveStatus, 1)
	group.Go(func() error {
		return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, ch)
	})
	return ch
}

func (s *Build) defaultExporter() []buildkit.ExportEntry {
	exp := buildkit.ExportEntry{
		Type:  buildkit.ExporterImage,
		Attrs: map[string]string{},
	}
	if len(s.Tag) > 0 {
		tags := s.Tag[:]
		for i, tag := range tags {
			ref, err := reference.ParseNormalizedNamed(tag)
			if err != nil {
				logrus.Warnf("Failed to normalize tag `%s` => %v", tag, err)
				continue
			}
			tags[i] = ref.String()
		}
		exp.Attrs["name"] = strings.Join(tags, ",")
		exp.Attrs["name-canonical"] = "" // true
	}
	return []buildkit.ExportEntry{exp}
}
