//go:build darwin

package daemon

func NewDefaultManager() Manager {
	manager, err := newDefaultLaunchdManager(directRunner{})
	if err != nil {
		return unavailableManager{err: err}
	}
	return manager
}
