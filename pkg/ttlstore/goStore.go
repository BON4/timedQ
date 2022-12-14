package ttlstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BON4/timedQ/pkg/coder"
)

const DEFAULT_DUMP_NAME = ".temp.db"

type MapEntity[K string, V any] struct {
	Key K
	Val V
}

type MapStore[K string, V any] struct {
	wg       *sync.WaitGroup
	cancel   context.CancelFunc
	store    *sync.Map
	ctx      context.Context
	save     chan MapEntity[K, TTLStoreEntity[V]]
	cfg      TTLStoreConfig
	dump     *os.File
	dumpPath string
}

// runSaveDaemon - saves data to file, stops after closed channel encountered
func runSaveDaemon[K string, V any](kv chan MapEntity[K, TTLStoreEntity[V]], wg *sync.WaitGroup, file io.Writer) {
	wg.Add(1)

	defer wg.Done()

	encoder := coder.NewEncoder[MapEntity[K, TTLStoreEntity[V]]](file)

	for {
		select {
		case data, ok := <-kv:
			if !ok {
				return
			}

			if err := encoder.Encode(&data); err != nil {
				//TODO: Log here
				panic(err)
			}
		}
	}
}

func runGcDaemon[K string, V any](ctx context.Context, store *sync.Map, wg *sync.WaitGroup, dRt time.Duration) {
	wg.Add(1)

	defer wg.Done()

	tiker := time.NewTicker(dRt)
	for {
		select {
		case <-tiker.C:
			store.Range(func(k, v any) bool {
				if val, ok := v.(TTLStoreEntity[V]); ok {
					eTime := val.GetTTL()
					if !(eTime <= 0) && eTime < time.Now().Unix() {
						store.Delete(k)
					}
				}
				// else {
				// 	// TODO: Make proper logger
				// 	fmt.Println("Invalid type value found while saving")
				// }
				return true
			})
		case <-ctx.Done():
			//TODO: log here
			return
		}
	}
}

// TODO: handle error
func NewMapStore[K string, V any](ctx context.Context, cfg TTLStoreConfig) *MapStore[K, V] {
	msctx, cancel := context.WithCancel(ctx)

	ms := &MapStore[K, V]{
		store:  &sync.Map{},
		ctx:    msctx,
		cancel: cancel,
		//TODO: CHANEL SIZE?
		save: make(chan MapEntity[K, TTLStoreEntity[V]], 100),
		cfg:  cfg,
		dump: nil,
		wg:   &sync.WaitGroup{},
	}

	dir, fname := filepath.Split(cfg.SavePath)
	if len(fname) == 0 {
		ms.dumpPath = dir + DEFAULT_DUMP_NAME
	} else {
		ms.dumpPath = cfg.SavePath
	}

	go runGcDaemon[K, V](ms.ctx, ms.store, ms.wg, ms.cfg.GCRefresh)
	return ms
}

func (ms *MapStore[K, V]) Path() string {
	return ms.dumpPath
}

// Run - runs a damon that saves map content to file
// Call Run, only in case of where cfg.MapStore.Save == true
// WRRNING: Load shoud be called before Run
func (ms *MapStore[K, V]) Run() error {
	if ms.cfg.Save {
		var err error
		ms.dump, err = os.OpenFile(ms.dumpPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println(err)
			return err
		}

		go runSaveDaemon[K, V](ms.save, ms.wg, ms.dump)
	}

	return nil
}

func (ms *MapStore[K, V]) Close() error {
	//TODO: Log here

	//stops gc daemon
	//and prevents Set method
	ms.cancel()

	//this will stop save daemon
	close(ms.save)

	//wait for daemons
	ms.wg.Wait()
	if ms.cfg.Save {
		//TODO: Error when close without run
		return ms.dump.Close()
	}
	return nil
}

// Load - loads all contents from file to internal map, then clears a file and dump all contents to fresh file
// WRRNING: Load shoud be called before Run
func (ms *MapStore[K, V]) Load() error {
	// Using custom split function we will get output in bytes where:
	// FIRST SCAN:
	// *******@mapEntity
	// ^_____^ - this is pre_separator (always len of 7)
	//
	// SECOND SCAN:
	// separator ... pre_sepator
	// ..............^__________ - now pre_separator will be at the end, we need to trim it from end and append it to start.
	if ms.cfg.Save {
		var err error
		reader, err := os.OpenFile(ms.dumpPath, os.O_CREATE|os.O_RDONLY, 0666)
		if err != nil {
			return err
		}

		decoder := coder.NewDecoder[MapEntity[K, TTLStoreEntity[V]]](reader)
		if err := decoder.Decode(func(ent *MapEntity[K, TTLStoreEntity[V]]) {
			ms.store.Store(ent.Key, ent.Val)
		}); err != nil {
			return err
		}

		if err := reader.Close(); err != nil {
			return err
		}

		writer, err := os.OpenFile(ms.dumpPath, os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}

		defer writer.Close()
		encoder := coder.NewEncoder[MapEntity[K, TTLStoreEntity[V]]](writer)

		ms.store.Range(func(key any, val any) bool {
			if okKey, ok := key.(K); ok {
				if okVal, ok := val.(TTLStoreEntity[V]); ok {
					if err = encoder.Encode(&MapEntity[K, TTLStoreEntity[V]]{
						Key: okKey,
						Val: okVal,
					}); err != nil {
						//TODO: Propper logger
						fmt.Printf("Error while updating storage file: %s\n", err.Error())
						return false
					}
				}
			}
			return true
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *MapStore[K, V]) Set(_ context.Context, key K, val V, ttl time.Duration) error {
	var t int64 = -1
	if ttl == 0 {
		return nil
	} else if ttl > 0 {
		t = time.Now().Add(ttl).Unix()
	}

	se := TTLStoreEntity[V]{
		Entity: val,
	}

	se.SetTTL(t)
	ms.store.Store(key, se)

	if ms.cfg.Save && ms.ctx.Err() == nil {
		ms.save <- MapEntity[K, TTLStoreEntity[V]]{Key: key, Val: se}
	}

	return nil
}

func (ms *MapStore[K, V]) Get(_ context.Context, key K) (V, bool) {
	var ent TTLStoreEntity[V]
	if val, ok := ms.store.Load(key); ok {
		if ent, ok := val.(TTLStoreEntity[V]); ok {
			eTime := ent.GetTTL()
			// 0 | 0 -> 0
			// 1 | 0 -> 1
			// 0 | 1 -> 1
			// 1 | 1 -> 1
			//fmt.Printf("Found with Get: %v\n", ent.Entity)
			if eTime > time.Now().Unix() || (eTime <= 0) {
				return ent.Entity, true
			} else {
				panic(fmt.Sprintf("Cant assert, got: %+v", val))
			}
		}
	}
	return ent.Entity, false
}

func (ms *MapStore[K, V]) Range(f func(key K, val V) bool) {
	ms.store.Range(func(key any, value any) bool {
		if okKey, ok := key.(K); ok {
			if okVal, ok := value.(TTLStoreEntity[V]); ok {
				return f(okKey, okVal.Entity)
			}
		}
		return true
	})
}
