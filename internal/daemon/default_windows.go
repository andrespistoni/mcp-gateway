//go:build windows

package daemon

func NewDefaultManager() Manager {
	manager, err := newDefaultTaskManager(directRunner{})
	if err != nil {
		return unavailableManager{err: err}
	}
	return manager
}
