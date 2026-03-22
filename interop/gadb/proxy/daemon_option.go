package proxy

// DaemonOption configures a Daemon.
type DaemonOption interface {
	applyDaemon(*daemonConfig)
}

// DaemonOptions is a slice of DaemonOption.
type DaemonOptions []DaemonOption

func (opts DaemonOptions) config() daemonConfig {
	cfg := defaultDaemonConfig()
	for _, o := range opts {
		o.applyDaemon(&cfg)
	}
	return cfg
}

type daemonConfig struct {
	ListenAddr string
	MapSize    uint32
}

func defaultDaemonConfig() daemonConfig {
	return daemonConfig{
		ListenAddr: ":7100",
		MapSize:    4 * 1024 * 1024,
	}
}

type daemonOptionListenAddr string

func (o daemonOptionListenAddr) applyDaemon(c *daemonConfig) { c.ListenAddr = string(o) }

// DaemonOptionListenAddr sets the TCP address the daemon listens on.
func DaemonOptionListenAddr(addr string) DaemonOption { return daemonOptionListenAddr(addr) }

type daemonOptionMapSize uint32

func (o daemonOptionMapSize) applyDaemon(c *daemonConfig) { c.MapSize = uint32(o) }

// DaemonOptionMapSize sets the mmap size for the binder driver.
func DaemonOptionMapSize(size uint32) DaemonOption { return daemonOptionMapSize(size) }
