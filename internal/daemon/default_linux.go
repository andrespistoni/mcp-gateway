//go:build linux

package daemon

func NewDefaultManager() Manager {
	manager, err := newDefaultSystemdManager(directRunner{})
	if err != nil {
		return unavailableManager{err: err}
	}
	return manager
}
