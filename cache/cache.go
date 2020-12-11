package cache

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/photostorm/ble"
)

type gattCache struct {
	filename string
	sync.RWMutex
}

func New(filename string) ble.GattCache {
	gc := gattCache{
		filename: filename,
	}

	return &gc
}

func (gc *gattCache) Store(mac ble.Addr, profile ble.Profile, replace bool) error {
	gc.Lock()
	defer gc.Unlock()

	cache, err := gc.loadExisting()
	if err != nil {
		return err
	}

	_, ok := cache[mac.String()]
	if ok && !replace {
		return fmt.Errorf("cache already contains gatt db for %s", mac.String())
	}

	cache[mac.String()] = profile

	err = gc.storeCache(cache)
	if err != nil {
		return err
	}

	return nil
}

func (gc *gattCache) Load(mac ble.Addr) (ble.Profile, error) {
	gc.RLock()
	defer gc.RUnlock()

	cache, err := gc.loadExisting()
	if err != nil {
		return ble.Profile{}, err
	}

	p, ok := cache[mac.String()]
	if !ok {
		return ble.Profile{}, fmt.Errorf("gatt db for %s not found in cache", mac.String())
	}

	return p, nil
}

func (gc *gattCache) Clear() error {
	gc.Lock()
	defer gc.Unlock()

	err := os.Remove(gc.filename)
	if err != nil {
		return err
	}

	return nil
}

func (gc *gattCache) loadExisting() (map[string]ble.Profile, error) {
	return nil, errors.New("disabled")
}

func (gc *gattCache) storeCache(cache map[string]ble.Profile) error {
	return errors.New("disabled")
}
