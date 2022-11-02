//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

// +build gofuzz

package moss

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/mschoch/smat"
)

// TODO: Test pre-allocated batches and AllocSet/Del/Merge().

var smatDebug = false

var smatCompactionSync = true

var smatCompactionConcern = CompactionAllow

// ------------------------------------------------

func smatLog(prefix, format string, args ...interface{}) {
	if smatDebug {
		fmt.Print(prefix)
		fmt.Printf(format, args...)
	}
}

// fuzz test using state machine driven by byte stream.
func Fuzz(data []byte) int {
	return smat.Fuzz(&smatContext{}, smat.ActionID('S'), smat.ActionID('T'),
		actionMap, data)
}

type smatContext struct {
	tmpDir string

	coll       Collection // Initialized in setupFunc().
	collMirror mirrorColl

	mo MergeOperator

	curBatch    int
	curSnapshot int
	curIterator int
	curKey      int

	batches      []Batch
	batchMirrors []map[string]smatBatchOp // Mirrors the entries in batches.

	snapshots       []Snapshot
	snapshotMirrors []*mirrorColl

	iterators       []Iterator
	iteratorMirrors []*mirrorIter

	keys []string

	actions int
}

type smatBatchOp struct {
	op, v string
}

type mirrorColl struct { // Used to validate coll and snapshot entries.
	kvs  map[string]string
	keys []string // Will be nil unless this is a snapshot.
}

type mirrorIter struct { // Used to validate iterator entries.
	pos int
	ss  *mirrorColl
}

// ------------------------------------------------------------------

var actionMap = smat.ActionMap{
	smat.ActionID('.'): action("      +batch", delta(func(c *smatContext) { c.curBatch++ })),
	smat.ActionID(','): action("      -batch", delta(func(c *smatContext) { c.curBatch-- })),
	smat.ActionID('{'): action("      +snapshot", delta(func(c *smatContext) { c.curSnapshot++ })),
	smat.ActionID('}'): action("      -snapshot", delta(func(c *smatContext) { c.curSnapshot-- })),
	smat.ActionID('['): action("      +itr", delta(func(c *smatContext) { c.curIterator++ })),
	smat.ActionID(']'): action("      -itr", delta(func(c *smatContext) { c.curIterator-- })),
	smat.ActionID(':'): action("      +key", delta(func(c *smatContext) { c.curKey++ })),
	smat.ActionID(';'): action("      -key", delta(func(c *smatContext) { c.curKey-- })),
	smat.ActionID('s'): action("    set", opSetFunc),
	smat.ActionID('d'): action("    del", opDelFunc),
	smat.ActionID('m'): action("    merge", opMergeFunc),
	smat.ActionID('g'): action("    get", opGetFunc),
	smat.ActionID('B'): action("  batchCreate", batchCreateFunc),
	smat.ActionID('b'): action("  batchExecute", batchExecuteFunc),
	smat.ActionID('H'): action("  snapshotCreate", snapshotCreateFunc),
	smat.ActionID('h'): action("  snapshotClose", snapshotCloseFunc),
	smat.ActionID('I'): action("  itrCreate", iteratorCreateFunc),
	smat.ActionID('i'): action("  itrClose", iteratorCloseFunc),
	smat.ActionID('>'): action("  itrNext", iteratorNextFunc),
	smat.ActionID('K'): action("  keyRegister", keyRegisterFunc),
	smat.ActionID('k'): action("  keyUnregister", keyUnregisterFunc),
	smat.ActionID('$'): action("CLOSE-REOPEN", closeReopenFunc),
}

var runningPercentActions []smat.PercentAction

func init() {
	var ids []int
	for actionId := range actionMap {
		ids = append(ids, int(actionId))
	}
	sort.Ints(ids)

	pct := 100 / len(actionMap)
	for _, actionId := range ids {
		runningPercentActions = append(runningPercentActions,
			smat.PercentAction{Percent: pct, Action: smat.ActionID(actionId)})
	}

	actionMap[smat.ActionID('S')] = action("SETUP", setupFunc)
	actionMap[smat.ActionID('T')] = action("TEARDOWN", teardownFunc)
}

// We only have one state: running.
func running(next byte) smat.ActionID {
	return smat.PercentExecute(next, runningPercentActions...)
}

// Creates an action func based on a callback, used for moving the curXxxx properties.
func delta(cb func(c *smatContext)) func(ctx smat.Context) (next smat.State, err error) {
	return func(ctx smat.Context) (next smat.State, err error) {
		c := ctx.(*smatContext)
		cb(c)
		if c.curBatch < 0 {
			c.curBatch = 1000
		}
		if c.curSnapshot < 0 {
			c.curSnapshot = 1000
		}
		if c.curIterator < 0 {
			c.curIterator = 1000
		}
		if c.curKey < 0 {
			c.curKey = 1000
		}
		return running, nil
	}
}

func action(name string, f func(ctx smat.Context) (smat.State, error)) func(ctx smat.Context) (smat.State, error) {
	return func(ctx smat.Context) (smat.State, error) {
		c := ctx.(*smatContext)
		c.actions++

		smatLog("  ", "%s\n", name)

		return f(ctx)
	}
}

// ------------------------------------------------------------------

var prefix = "                          "

func setupFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)

	c.tmpDir, err = ioutil.TempDir("", "mossStoreSMAT")
	if err != nil {
		return nil, err
	}

	next, err = closeReopenFunc(ctx)
	if err != nil {
		return next, err
	}

	c.collMirror.kvs = map[string]string{}

	return running, nil
}

func teardownFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)

	err = closeParts(c)

	if c.tmpDir != "" {
		os.RemoveAll(c.tmpDir)
	}

	return nil, err
}

// ------------------------------------------------------------------

func opSetFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	b, mirror, err := c.getCurBatch()
	if err != nil {
		return nil, err
	}
	k := c.getCurKey()
	_, exists := mirror[k]
	if !exists {
		v := fmt.Sprintf("%s-%d", k, c.actions)
		mirror[k] = smatBatchOp{op: "set", v: v}
		err := b.Set([]byte(k), []byte(v))
		if err != nil {
			return nil, err
		}
	}
	return running, nil
}

func opDelFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	b, mirror, err := c.getCurBatch()
	if err != nil {
		return nil, err
	}
	k := c.getCurKey()
	_, exists := mirror[k]
	if !exists {
		mirror[k] = smatBatchOp{op: "del"}
		err := b.Del([]byte(k))
		if err != nil {
			return nil, err
		}
	}
	return running, nil
}

func opMergeFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	b, mirror, err := c.getCurBatch()
	if err != nil {
		return nil, err
	}
	k := c.getCurKey()
	_, exists := mirror[k]
	if !exists {
		v := fmt.Sprintf("%s-%d", k, c.actions)
		mirror[k] = smatBatchOp{op: "merge", v: v}
		err := b.Merge([]byte(k), []byte(v))
		if err != nil {
			return nil, err
		}
	}
	return running, nil
}

func opGetFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	ss, ssMirror, err := c.getCurSnapshot()
	if err != nil {
		return nil, err
	}
	k := c.getCurKey()
	v, err := ss.Get([]byte(k), ReadOptions{})
	if err != nil {
		return nil, err
	}
	mirrorV := ssMirror.kvs[k]
	if string(v) != mirrorV {
		return nil, fmt.Errorf("get mismatch, got: %s, mirror: %s", v, mirrorV)
	}
	return running, nil
}

func batchCreateFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	b, err := c.coll.NewBatch(0, 0)
	if err != nil {
		return nil, err
	}
	c.batches = append(c.batches, b)
	c.batchMirrors = append(c.batchMirrors, map[string]smatBatchOp{})
	return running, nil
}

func batchExecuteFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	if len(c.batches) <= 0 {
		return running, nil
	}
	b := c.batches[c.curBatch%len(c.batches)]
	err = c.coll.ExecuteBatch(b, WriteOptions{})
	if err != nil {
		return nil, err
	}
	err = b.Close()
	if err != nil {
		return nil, err
	}
	smatBatchOps := c.batchMirrors[c.curBatch%len(c.batchMirrors)]
	for key, smatBatchOp := range smatBatchOps {
		smatLog("            ", "%s = %+v\n", key, smatBatchOp)
		if smatBatchOp.op == "set" {
			c.collMirror.kvs[key] = smatBatchOp.v
		} else if smatBatchOp.op == "del" {
			delete(c.collMirror.kvs, key)
		} else if smatBatchOp.op == "merge" {
			k := []byte(key)
			v := []byte(smatBatchOp.v)
			mv, ok := c.mo.FullMerge(k, []byte(c.collMirror.kvs[key]), [][]byte{v})
			if !ok {
				return nil, fmt.Errorf("failed FullMerge")
			}
			c.collMirror.kvs[key] = string(mv)
		} else {
			return nil, fmt.Errorf("unexpected smatBatchOp.op: %+v, key: %s", smatBatchOp, key)
		}
	}
	smatLog(prefix, "collMirror.kvs: %+v\n", c.collMirror.kvs)
	i := c.curBatch % len(c.batches)
	c.batches = append(c.batches[:i], c.batches[i+1:]...)
	i = c.curBatch % len(c.batchMirrors)
	c.batchMirrors = append(c.batchMirrors[:i], c.batchMirrors[i+1:]...)
	return running, nil
}

func snapshotCreateFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	ss, err := c.coll.Snapshot()
	smatLog(prefix, "snapshotCreateFunc, coll.Snapshot, ss: %p, %#v\n", ss, ss)
	if err != nil {
		return nil, err
	}
	smatLog(prefix, "snapshotCreate: %d\n", len(c.snapshots))

	itr, _ := ss.StartIterator(nil, nil, IteratorOptions{})
	i := 0
	for {
		ik, iv, err := itr.Current()
		smatLog(prefix, "snapshotCreate, test itr i: %d, k: %s, v: %s, err: %v\n", i, ik, iv, err)
		if err == ErrIteratorDone {
			break
		}
		err = itr.Next()
		if err == ErrIteratorDone {
			break
		}
		i++
	}
	itr.Close()

	c.snapshots = append(c.snapshots, ss)
	c.snapshotMirrors = append(c.snapshotMirrors, c.collMirror.snapshot())
	return running, nil
}

func snapshotCloseFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	if len(c.snapshots) <= 0 {
		return running, nil
	}
	if len(c.snapshots) != len(c.snapshotMirrors) {
		return running, fmt.Errorf("len snapshots != len snapshotMirrors")
	}
	i := c.curSnapshot % len(c.snapshots)
	smatLog(prefix, "curSnapshot: %d\n", i)
	ss := c.snapshots[i]
	ssm := c.snapshotMirrors[i]

	for j := len(c.iteratorMirrors) - 1; j >= 0; j-- { // Close any child iterators.
		itrm := c.iteratorMirrors[j]
		if itrm.ss == ssm {
			itr := c.iterators[j]
			err = itr.Close()
			if err != nil {
				return nil, err
			}
			c.iterators = append(c.iterators[:j], c.iterators[j+1:]...)
			c.iteratorMirrors = append(c.iteratorMirrors[:j], c.iteratorMirrors[j+1:]...)
		}
	}

	err = ss.Close()
	if err != nil {
		return nil, err
	}
	c.snapshots = append(c.snapshots[:i], c.snapshots[i+1:]...)
	c.snapshotMirrors = append(c.snapshotMirrors[:i], c.snapshotMirrors[i+1:]...)
	smatLog(prefix, "snapshots left: %d\n", len(c.snapshots))
	return running, nil
}

func iteratorCreateFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	ss, ssMirror, err := c.getCurSnapshot()
	if err != nil {
		return nil, err
	}
	iter, err := ss.StartIterator(nil, nil, IteratorOptions{})
	if err != nil {
		return nil, err
	}
	smatLog(prefix, "iteratorCreate: %d\n", len(c.iterators))
	c.iterators = append(c.iterators, iter)
	c.iteratorMirrors = append(c.iteratorMirrors, &mirrorIter{ss: ssMirror})
	return running, nil
}

func iteratorCloseFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	if len(c.iterators) <= 0 {
		return running, nil
	}
	i := c.curIterator % len(c.iterators)
	iter := c.iterators[i]
	err = iter.Close()
	if err != nil {
		return nil, err
	}
	c.iterators = append(c.iterators[:i], c.iterators[i+1:]...)
	c.iteratorMirrors = append(c.iteratorMirrors[:i], c.iteratorMirrors[i+1:]...)
	return running, nil
}

func iteratorNextFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	if len(c.iterators) <= 0 {
		return running, nil
	}
	if len(c.iterators) != len(c.iteratorMirrors) {
		return running, fmt.Errorf("len iterators != len iteratorMirrors")
	}
	smatLog(prefix, "numIterators: %d\n", len(c.iterators))

	iterIdx := c.curIterator % len(c.iterators)
	iter := c.iterators[iterIdx]
	iterMirror := c.iteratorMirrors[iterIdx]

	smatLog(prefix, "iteratorNext: %p, %#v\n", iter, iter)

	iteratorActual, ok := iter.(*iterator)
	if ok {
		smatLog(prefix, "iteratorNext.ss: %p, %#v\n", iteratorActual.ss, iteratorActual.ss)
		smatLog(prefix, "iteratorNext.llIter: %#v\n", iteratorActual.lowerLevelIter)
		lli, ok := iteratorActual.lowerLevelIter.(*iterator)
		if ok {
			smatLog(prefix, "iteratorNext.llIter: %#v\n", lli)

			iss := lli.ss
			itr, _ := iss.StartIterator(nil, nil, IteratorOptions{})
			smatLog(prefix, "iteratorNext check itr: %p, %#v\n", itr, itr)
			i := 0
			for {
				var ik, iv []byte
				ik, iv, err = itr.Current()
				smatLog(prefix, "== iteratorNext, check itr i: %d, k: %s, v: %s, err: %v\n", i, ik, iv, err)
				if err == ErrIteratorDone {
					break
				}
				err = itr.Next()
				if err == ErrIteratorDone {
					break
				}
				i++
			}
			itr.Close()
		}
	}

	err = iter.Next()
	iterMirror.pos++
	smatLog(prefix, "iterIdx: %d, iterMirror.pos: %d, iterMirror.iter.ss: %+v\n",
		iterIdx, iterMirror.pos, iterMirror.ss)
	if err != nil && err != ErrIteratorDone {
		return nil, err
	}
	k, v, err := iter.Current()
	if err != nil {
		if err != ErrIteratorDone {
			return nil, err
		}
		if iterMirror.pos < len(iterMirror.ss.keys) {
			return nil, fmt.Errorf("iter done but iterMirror not done")
		}
	} else {
		if iterMirror.pos >= len(iterMirror.ss.keys) {
			return nil, fmt.Errorf("iterMirror done but iter not done")
		}
		iterMirrorKey := iterMirror.ss.keys[iterMirror.pos]
		if string(k) != iterMirrorKey {
			return nil, fmt.Errorf("iterMirror key != iter key")
		}
		if string(v) != iterMirror.ss.kvs[iterMirrorKey] {
			return nil, fmt.Errorf("iter val (%q) != iterMirror val (%q)",
				string(v), iterMirror.ss.kvs[iterMirrorKey])
		}
	}
	return running, nil
}

func keyRegisterFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	c.keys = append(c.keys, fmt.Sprintf("%d", c.curKey+len(c.keys)))
	return running, nil
}

func keyUnregisterFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)
	if len(c.keys) <= 0 {
		return running, nil
	}
	i := c.curKey % len(c.keys)
	c.keys = append(c.keys[:i], c.keys[i+1:]...)
	return running, nil
}

// ------------------------------------------------------

func closeReopenFunc(ctx smat.Context) (next smat.State, err error) {
	c := ctx.(*smatContext)

	waitUntilClean := func() error {
		for c.coll != nil &&
			c.coll.(*collection) != nil { // Wait until dirty ops are drained.
			var stats *CollectionStats
			stats, err = c.coll.Stats()
			if err != nil {
				return err
			}

			if stats.CurDirtyOps <= 0 &&
				stats.CurDirtyBytes <= 0 &&
				stats.CurDirtySegments <= 0 {
				break
			}

			smatLog(prefix, "notifying merger, CurDirtyOps: %d\n", stats.CurDirtyOps)

			c.coll.(*collection).NotifyMerger("mergeAll", true)

			time.Sleep(200 * time.Millisecond)
		}

		return nil
	}

	err = waitUntilClean()
	if err != nil {
		return nil, err
	}

	err = closeParts(c)
	if err != nil {
		return nil, err
	}

	err = waitUntilClean()
	if err != nil {
		return nil, err
	}

	co := CollectionOptions{
		MergeOperator: &MergeOperatorStringAppend{Sep: ":"},
	}

	compactionSync := smatCompactionSync

	store, err := OpenStore(c.tmpDir, StoreOptions{
		CollectionOptions: co,
		CompactionSync:    compactionSync,
	})
	if err != nil {
		return nil, err
	}

	storeSnapshotInit, err := store.Snapshot()
	smatLog(prefix, "closeReopenFunc, storeSnapshotInit, ss: %p\n", storeSnapshotInit)
	if err != nil {
		return nil, err
	}

	smatLog(prefix, "storeSnapshotInit: %+v\n", storeSnapshotInit)

	co.LowerLevelInit = storeSnapshotInit
	co.LowerLevelUpdate = func(higher Snapshot) (Snapshot, error) {
		smatLog(prefix, "LowerLevelUpdate... higher: %+v\n", higher)
		smatLog(prefix, "LowerLevelUpdate... higher.a: %+v\n", higher.(*segmentStack).a)

		var ss Snapshot
		ss, err = store.Persist(higher, StorePersistOptions{
			CompactionConcern: smatCompactionConcern,
		})

		if err != nil {
			smatLog(prefix, "LowerLevelUpdate, err: %v\n", err)
		} else {
			smatLog(prefix, "LowerLevelUpdate, after persist, footer: %+v\n", ss.(*Footer))
		}

		return ss, err
	}
	co.OnEvent = func(ev Event) {
		if ev.Kind == EventKindClose {
			store.Close()
		}
	}

	coll, err := NewCollection(co)
	if err != nil {
		return nil, err
	}

	err = coll.Start()
	if err != nil {
		return nil, err
	}

	smatLog(prefix, "closeReopen coll: %+v\n", coll)
	ss, _ := coll.Snapshot()
	smatLog(prefix, "closeReopenFunc, coll.Snapshot, ss: %p\n", ss)

	v, err := ss.Get([]byte("2"), ReadOptions{})
	smatLog(prefix, "coll.Get(2), v: %s, err: %v\n", v, err)
	itr, _ := ss.StartIterator(nil, nil, IteratorOptions{})
	i := 0
	for {
		ik, iv, err := itr.Current()
		smatLog(prefix, "coll.Iterator().current, i: %d, k: %s, v: %s, err: %v\n", i, ik, iv, err)
		if err == ErrIteratorDone {
			break
		}
		err = itr.Next()
		if err == ErrIteratorDone {
			break
		}
		i++
	}
	itr.Close()
	ss.Close()

	c.coll = coll
	c.mo = co.MergeOperator

	return running, nil
}

func closeParts(c *smatContext) (err error) {
	for _, iter := range c.iterators {
		err = iter.Close()
		if err != nil {
			return err
		}
	}
	c.iterators = nil
	c.iteratorMirrors = nil

	for _, ss := range c.snapshots {
		err = ss.Close()
		if err != nil {
			return err
		}
	}
	c.snapshots = nil
	c.snapshotMirrors = nil

	for _, b := range c.batches {
		err = b.Close()
		if err != nil {
			return err
		}
	}
	c.batches = nil
	c.batchMirrors = nil

	if c.coll != nil {
		err = c.coll.Close()
		if err != nil {
			return err
		}
		c.coll = nil
	}

	return nil
}

// ------------------------------------------------------

func (c *smatContext) getCurKey() string {
	if len(c.keys) <= 0 {
		return "x"
	}
	return c.keys[c.curKey%len(c.keys)]
}

func (c *smatContext) getCurBatch() (Batch, map[string]smatBatchOp, error) {
	if len(c.batches) <= 0 {
		_, err := batchCreateFunc(c)
		if err != nil {
			return nil, nil, err
		}
	}
	return c.batches[c.curBatch%len(c.batches)],
		c.batchMirrors[c.curBatch%len(c.batchMirrors)], nil
}

func (c *smatContext) getCurSnapshot() (Snapshot, *mirrorColl, error) {
	if len(c.snapshots) <= 0 {
		smatLog(prefix, "==== getCurSnapshot.snapshotCreateFunc\n")
		_, err := snapshotCreateFunc(c)
		if err != nil {
			return nil, nil, err
		}
	}
	return c.snapshots[c.curSnapshot%len(c.snapshots)],
		c.snapshotMirrors[c.curSnapshot%len(c.snapshotMirrors)], nil
}

// ------------------------------------------------------------------

func (mc *mirrorColl) snapshot() *mirrorColl {
	kvs := map[string]string{}
	keys := make([]string, 0, len(kvs))
	for k, v := range mc.kvs {
		kvs[k] = v
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &mirrorColl{kvs: kvs, keys: keys}
}
