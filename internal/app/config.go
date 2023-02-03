// Package app implements functionality related to reading and writing app
// configuration files.
package app

import (
	"github.com/superfly/flyctl/api"
	"github.com/superfly/flyctl/scanner"
)

const (
	// DefaultConfigFileName denotes the default application configuration file name.
	DefaultConfigFileName = "fly.toml"
	// Config is versioned, initially, to separate nomad from machine apps without having to consult
	// the API
	NomadPlatform    = "nomad"
	MachinesPlatform = "machines"
)

func NewConfig() *Config {
	return &Config{
		Definition: map[string]any{},
	}
}

// Config wraps the properties of app configuration.
type Config struct {
	AppName       string                    `toml:"app,omitempty" json:"app,omitempty"`
	KillSignal    string                    `toml:"kill_signal,omitempty" json:"kill_signal,omitempty"`
	KillTimeout   int                       `toml:"kill_timeout,omitempty" json:"kill_timeout,omitempty"`
	Build         *Build                    `toml:"build,omitempty" json:"build,omitempty"`
	HttpService   *HTTPService              `toml:"http_service,omitempty" json:"http_service,omitempty"`
	Services      []Service                 `toml:"services" json:"services,omitempty"`
	Env           map[string]string         `toml:"env" json:"env,omitempty"`
	Metrics       *api.MachineMetrics       `toml:"metrics" json:"metrics,omitempty"`
	Statics       []Static                  `toml:"statics,omitempty" json:"statics,omitempty"`
	Deploy        *Deploy                   `toml:"deploy, omitempty" json:"deploy,omitempty"`
	PrimaryRegion string                    `toml:"primary_region,omitempty" json:"primary_region,omitempty"`
	Checks        map[string]*ToplevelCheck `toml:"checks,omitempty" json:"checks,omitempty"`
	Mounts        *Volume                   `toml:"mounts,omitempty" json:"mounts,omitempty"`
	Processes     map[string]string         `toml:"processes,omitempty" json:"processes,omitempty"`
	Experimental  *Experimental             `toml:"experimental,omitempty" json:"experimental,omitempty"`

	FlyTomlPath string         `toml:"-" json:"-"`
	Definition  map[string]any `toml:"-" json:"-"`
}

type Deploy struct {
	ReleaseCommand string `toml:"release_command,omitempty"`
	Strategy       string `toml:"strategy,omitempty"`
}

type Static struct {
	GuestPath string `toml:"guest_path" json:"guest_path" validate:"required"`
	UrlPrefix string `toml:"url_prefix" json:"url_prefix" validate:"required"`
}

type Volume struct {
	Source      string `toml:"source" json:"source"`
	Destination string `toml:"destination" json:"destination"`
}

type VM struct {
	CpuCount int `toml:"cpu_count,omitempty"`
	Memory   int `toml:"memory,omitempty"`
}

type Build struct {
	Builder           string            `toml:"builder,omitempty"`
	Args              map[string]string `toml:"args,omitempty"`
	Buildpacks        []string          `toml:"buildpacks,omitempty"`
	Image             string            `toml:"image,omitempty"`
	Settings          map[string]any    `toml:"settings,omitempty"`
	Builtin           string            `toml:"builtin,omitempty"`
	Dockerfile        string            `toml:"dockerfile,omitempty"`
	Ignorefile        string            `toml:"ignorefile,omitempty"`
	DockerBuildTarget string            `toml:"build-target,omitempty"`
}

type Experimental struct {
	Cmd          []string `toml:"cmd,omitempty"`
	Entrypoint   []string `toml:"entrypoint,omitempty"`
	Exec         []string `toml:"exec,omitempty"`
	AutoRollback bool     `toml:"auto_rollback,omitempty"`
	EnableConsul bool     `toml:"enable_consul,omitempty"`
	EnableEtcd   bool     `toml:"enable_etcd,omitempty"`
}

func (c *Config) HasNonHttpAndHttpsStandardServices() bool {
	for _, service := range c.Services {
		switch service.Protocol {
		case "udp":
			return true
		case "tcp":
			for _, p := range service.Ports {
				if p.HasNonHttpPorts() {
					return true
				} else if p.ContainsPort(80) && (len(p.Handlers) != 1 || p.Handlers[0] != "http") {
					return true
				} else if p.ContainsPort(443) && (len(p.Handlers) != 2 || p.Handlers[0] != "tls" || p.Handlers[1] != "http") {
					return true
				}
			}
		}
	}
	return false
}

func (c *Config) HasUdpService() bool {
	for _, service := range c.Services {
		if service.Protocol == "udp" {
			return true
		}
	}
	return false
}

func (c *Config) Dockerfile() string {
	if c.Build == nil {
		return ""
	}
	return c.Build.Dockerfile
}

func (c *Config) Ignorefile() string {
	if c.Build == nil {
		return ""
	}
	return c.Build.Ignorefile
}

func (c *Config) DockerBuildTarget() string {
	if c.Build == nil {
		return ""
	}
	return c.Build.DockerBuildTarget
}

func (c *Config) SetInternalPort(port int) bool {
	if len(c.Services) > 0 {
		c.Services[0].InternalPort = port
		return true
	}
	return false
}

func (c *Config) SetConcurrency(soft int, hard int) bool {
	if len(c.Services) == 0 {
		return false
	}

	service := &c.Services[0]
	if service.Concurrency == nil {
		service.Concurrency = &api.MachineServiceConcurrency{}
	}
	service.Concurrency.HardLimit = hard
	service.Concurrency.SoftLimit = soft
	return true
}

func (c *Config) SetReleaseCommand(cmd string) {
	if c.Deploy == nil {
		c.Deploy = &Deploy{}
	}
	c.Deploy.ReleaseCommand = cmd
}

func (c *Config) SetDockerCommand(cmd string) {
	if c.Experimental == nil {
		c.Experimental = &Experimental{}
	}
	c.Experimental.Cmd = []string{cmd}
}

func (c *Config) SetKillSignal(signal string) {
	c.KillSignal = signal
}

func (c *Config) SetDockerEntrypoint(entrypoint string) {
	if c.Experimental == nil {
		c.Experimental = &Experimental{}
	}
	c.Experimental.Entrypoint = []string{entrypoint}
}

func (c *Config) SetEnvVariable(name, value string) {
	c.Env[name] = value
}

func (c *Config) SetEnvVariables(vals map[string]string) {
	for k, v := range vals {
		c.SetEnvVariable(k, v)
	}
}

func (c *Config) GetEnvVariables() map[string]string {
	return c.Env
}

func (c *Config) SetProcess(name, value string) {
	if c.Processes == nil {
		c.Processes = make(map[string]string)
	}
	c.Processes[name] = value
}

func (c *Config) SetStatics(statics []scanner.Static) {
	c.Statics = make([]Static, 0, len(statics))
	for _, static := range statics {
		c.Statics = append(c.Statics, Static{
			GuestPath: static.GuestPath,
			UrlPrefix: static.UrlPrefix,
		})
	}
}

func (c *Config) SetVolumes(volumes []scanner.Volume) {
	if len(volumes) == 0 {
		return
	}
	// FIXME: "mounts" section is confusing, it is plural but only allows one mount
	c.Mounts = &Volume{
		Source:      volumes[0].Source,
		Destination: volumes[0].Destination,
	}
}
