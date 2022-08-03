package connection

import "github.com/stackrox/rox/pkg/sync"

var (
	waitGroupMap  = make(map[string]*sync.WaitGroup)
	waitGroupLock sync.Mutex
)

func AddCheckpoint(uuid string, count int) {
	waitGroupLock.Lock()
	defer waitGroupLock.Unlock()

	var wg sync.WaitGroup
	wg.Add(count)
	waitGroupMap[uuid] = &wg
}

func MarkCheckpoint(uuid string) {
	waitGroupLock.Lock()
	defer waitGroupLock.Unlock()
	waitGroupMap[uuid].Done()
}

func WaitForCheckpoint(uuid string) {
	waitGroupLock.Lock()
	wg := waitGroupMap[uuid]
	if wg == nil {
		panic("nope not good")
	}
	waitGroupLock.Unlock()

	wg.Wait()

	waitGroupLock.Lock()
	delete(waitGroupMap, uuid)
	waitGroupLock.Unlock()
}
