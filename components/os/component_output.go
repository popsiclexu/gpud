package os

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/leptonai/gpud/components"
	components_metrics "github.com/leptonai/gpud/components/metrics"
	"github.com/leptonai/gpud/components/query"
	"github.com/leptonai/gpud/pkg/file"
	"github.com/leptonai/gpud/pkg/process"

	"github.com/shirou/gopsutil/v4/host"
	procs "github.com/shirou/gopsutil/v4/process"
)

type Output struct {
	Host                        Host     `json:"host"`
	Kernel                      Kernel   `json:"kernel"`
	Platform                    Platform `json:"platform"`
	Uptimes                     Uptimes  `json:"uptimes"`
	ProcessCountZombieProcesses int      `json:"process_count_zombie_processes"`
}

type Host struct {
	ID string `json:"id"`
}

type Kernel struct {
	Arch    string `json:"arch"`
	Version string `json:"version"`
}

type Platform struct {
	Name    string `json:"name"`
	Family  string `json:"family"`
	Version string `json:"version"`
}

type Uptimes struct {
	Seconds          uint64 `json:"seconds"`
	SecondsHumanized string `json:"seconds_humanized"`

	BootTimeUnixSeconds uint64 `json:"boot_time_unix_seconds"`
	BootTimeHumanized   string `json:"boot_time_humanized"`
}

func (o *Output) JSON() ([]byte, error) {
	return json.Marshal(o)
}

func ParseOutputJSON(data []byte) (*Output, error) {
	o := new(Output)
	if err := json.Unmarshal(data, o); err != nil {
		return nil, err
	}
	return o, nil
}

const (
	StateNameHost  = "host"
	StateKeyHostID = "id"

	StateNameKernel       = "kernel"
	StateKeyKernelArch    = "arch"
	StateKeyKernelVersion = "version"

	StateNamePlatform       = "platform"
	StateKeyPlatformName    = "name"
	StateKeyPlatformFamily  = "family"
	StateKeyPlatformVersion = "version"

	StateNameUptimes                   = "uptimes"
	StateKeyUptimesSeconds             = "uptime_seconds"
	StateKeyUptimesHumanized           = "uptime_humanized"
	StateKeyUptimesBootTimeUnixSeconds = "boot_time_unix_seconds"
	StateKeyUptimesBootTimeHumanized   = "boot_time_humanized"

	StateNameProcessCountsByStatus      = "process_counts_by_status"
	StateKeyProcessCountZombieProcesses = "process_count_zombie_processes"
)

func ParseStateHost(m map[string]string) (Host, error) {
	h := Host{}
	h.ID = m[StateKeyHostID]
	return h, nil
}

func ParseStateKernel(m map[string]string) (Kernel, error) {
	k := Kernel{}
	k.Arch = m[StateKeyKernelArch]
	k.Version = m[StateKeyKernelVersion]
	return k, nil
}

func ParseStatePlatform(m map[string]string) (Platform, error) {
	p := Platform{}
	p.Name = m[StateKeyPlatformName]
	p.Family = m[StateKeyPlatformFamily]
	p.Version = m[StateKeyPlatformVersion]
	return p, nil
}

func ParseStateUptimes(m map[string]string) (Uptimes, error) {
	u := Uptimes{}

	var err error
	u.Seconds, err = strconv.ParseUint(m[StateKeyUptimesSeconds], 10, 64)
	if err != nil {
		return Uptimes{}, err
	}
	u.SecondsHumanized = m[StateKeyUptimesHumanized]

	u.BootTimeUnixSeconds, err = strconv.ParseUint(m[StateKeyUptimesBootTimeUnixSeconds], 10, 64)
	if err != nil {
		return Uptimes{}, err
	}
	u.BootTimeHumanized = m[StateKeyUptimesBootTimeHumanized]

	return u, nil
}

func ParseStateProcessCountZombieProcesses(m map[string]string) (int, error) {
	s, ok := m[StateKeyProcessCountZombieProcesses]
	if ok {
		count, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return count, nil
	}
	return 0, nil
}

func ParseStatesToOutput(states ...components.State) (*Output, error) {
	o := &Output{}
	for _, state := range states {
		switch state.Name {
		case StateNameHost:
			host, err := ParseStateHost(state.ExtraInfo)
			if err != nil {
				return nil, err
			}
			o.Host = host

		case StateNameKernel:
			kernel, err := ParseStateKernel(state.ExtraInfo)
			if err != nil {
				return nil, err
			}
			o.Kernel = kernel

		case StateNamePlatform:
			platform, err := ParseStatePlatform(state.ExtraInfo)
			if err != nil {
				return nil, err
			}
			o.Platform = platform

		case StateNameUptimes:
			uptimes, err := ParseStateUptimes(state.ExtraInfo)
			if err != nil {
				return nil, err
			}
			o.Uptimes = uptimes

		case StateNameProcessCountsByStatus:
			var err error
			o.ProcessCountZombieProcesses, err = ParseStateProcessCountZombieProcesses(state.ExtraInfo)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("unknown state name: %s", state.Name)
		}
	}
	return o, nil
}

func (o *Output) States() ([]components.State, error) {
	states := []components.State{
		{
			Name:    StateNameHost,
			Healthy: true,
			Reason:  fmt.Sprintf("host id: %s", o.Host.ID),
			ExtraInfo: map[string]string{
				StateKeyHostID: o.Host.ID,
			},
		},
		{
			Name:    StateNameKernel,
			Healthy: true,
			Reason:  fmt.Sprintf("arch: %s, version: %s", o.Kernel.Arch, o.Kernel.Version),
			ExtraInfo: map[string]string{
				StateKeyKernelArch:    o.Kernel.Arch,
				StateKeyKernelVersion: o.Kernel.Version,
			},
		},
		{
			Name:    StateNamePlatform,
			Healthy: true,
			Reason:  fmt.Sprintf("name: %s, family: %s, version: %s", o.Platform.Name, o.Platform.Family, o.Platform.Version),
			ExtraInfo: map[string]string{
				StateKeyPlatformName:    o.Platform.Name,
				StateKeyPlatformFamily:  o.Platform.Family,
				StateKeyPlatformVersion: o.Platform.Version,
			},
		},
		{
			Name:    StateNameUptimes,
			Healthy: true,
			Reason:  fmt.Sprintf("uptime: %s, boot time: %s", o.Uptimes.SecondsHumanized, o.Uptimes.BootTimeHumanized),
			ExtraInfo: map[string]string{
				StateKeyUptimesSeconds:             fmt.Sprintf("%d", o.Uptimes.Seconds),
				StateKeyUptimesHumanized:           o.Uptimes.SecondsHumanized,
				StateKeyUptimesBootTimeUnixSeconds: fmt.Sprintf("%d", o.Uptimes.BootTimeUnixSeconds),
				StateKeyUptimesBootTimeHumanized:   o.Uptimes.BootTimeHumanized,
			},
		},
	}

	stateProcCounts := components.State{
		Name:    StateNameProcessCountsByStatus,
		Healthy: true,
		ExtraInfo: map[string]string{
			StateKeyProcessCountZombieProcesses: fmt.Sprintf("%d", o.ProcessCountZombieProcesses),
		},
	}
	if o.ProcessCountZombieProcesses >= DefaultZombieProcessCountThreshold {
		stateProcCounts.Healthy = false
		stateProcCounts.Reason = fmt.Sprintf("too many zombie processes: %d (threshold: %d)", o.ProcessCountZombieProcesses, DefaultZombieProcessCountThreshold)
	} else {
		stateProcCounts.Reason = fmt.Sprintf("zombie processes: %d (threshold: %d)", o.ProcessCountZombieProcesses, DefaultZombieProcessCountThreshold)
	}

	states = append(states, stateProcCounts)
	return states, nil
}

var DefaultZombieProcessCountThreshold = 1000

func init() {
	// e.g., "/proc/sys/fs/file-max" exists on linux
	if file.CheckFDLimitSupported() {
		limit, err := file.GetLimit()
		if limit > 0 && err == nil {
			// set to 20% of system limit
			DefaultZombieProcessCountThreshold = int(float64(limit) * 0.20)
		}
	}
}

var (
	defaultPollerOnce sync.Once
	defaultPoller     query.Poller
)

// only set once since it relies on the kube client and specific port
func setDefaultPoller(cfg Config) {
	defaultPollerOnce.Do(func() {
		defaultPoller = query.New(Name, cfg.Query, Get)
	})
}

func getDefaultPoller() query.Poller {
	return defaultPoller
}

func Get(ctx context.Context) (_ any, e error) {
	defer func() {
		if e != nil {
			components_metrics.SetGetFailed(Name)
		} else {
			components_metrics.SetGetSuccess(Name)
		}
	}()

	o := &Output{}

	hostID, err := host.HostID()
	if err != nil {
		return nil, err
	}
	o.Host = Host{ID: hostID}

	arch, err := host.KernelArch()
	if err != nil {
		return nil, err
	}
	kernelVer, err := host.KernelVersion()
	if err != nil {
		return nil, err
	}
	o.Kernel = Kernel{Arch: arch, Version: kernelVer}

	platform, family, version, err := host.PlatformInformation()
	if err != nil {
		return nil, err
	}
	o.Platform = Platform{Name: platform, Family: family, Version: version}

	uptime, err := host.UptimeWithContext(ctx)
	if err != nil {
		return nil, err
	}
	boottime, err := host.BootTimeWithContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	o.Uptimes = Uptimes{
		Seconds:             uptime,
		SecondsHumanized:    humanize.RelTime(now.Add(time.Duration(-int64(uptime))*time.Second), now, "ago", "from now"),
		BootTimeUnixSeconds: boottime,
		BootTimeHumanized:   humanize.RelTime(time.Unix(int64(boottime), 0), now, "ago", "from now"),
	}

	allProcs, err := process.CountProcessesByStatus(ctx)
	if err != nil {
		return nil, err
	}

	for status, procsWithStatus := range allProcs {
		if status == procs.Zombie {
			o.ProcessCountZombieProcesses = len(procsWithStatus)
			break
		}
	}

	return o, nil
}
