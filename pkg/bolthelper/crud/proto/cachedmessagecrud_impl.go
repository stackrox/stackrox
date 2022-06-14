package proto

import (
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/pkg/storecache"
)

type cachedMessageCrudImpl struct {
	messageCrud MessageCrud

	metricType string
	metricFunc func(string, string)

	readVersion uint64
	cache       storecache.Cache
}

func (c *cachedMessageCrudImpl) stringKey(msg proto.Message) string {
	return string(c.KeyFunc(msg))
}

func (c *cachedMessageCrudImpl) getReadVersion() uint64 {
	return atomic.LoadUint64(&c.readVersion)
}

func (c *cachedMessageCrudImpl) addReadVersion(delta uint64) {
	atomic.AddUint64(&c.readVersion, delta)
}

func (c *cachedMessageCrudImpl) Read(id string) (proto.Message, error) {
	if cached := c.cache.Get(id); cached != nil {
		c.metricFunc("hit", c.metricType)
		return proto.Clone(cached.(proto.Message)), nil
	}
	c.metricFunc("miss", c.metricType)
	readVersion := c.getReadVersion()
	msg, err := c.messageCrud.Read(id)
	if msg != nil {
		c.cache.Add(id, msg, readVersion)
	}
	return msg, err
}

func (c *cachedMessageCrudImpl) ReadBatch(ids []string) ([]proto.Message, []int, error) {
	var cachedMsgs []proto.Message
	var uncachedIds []string
	for _, id := range ids {
		if cached := c.cache.Get(id); cached != nil {
			c.metricFunc("hit", c.metricType)
			cachedMsgs = append(cachedMsgs, proto.Clone(cached.(proto.Message)))
		} else {
			c.metricFunc("miss", c.metricType)
			uncachedIds = append(uncachedIds, id)
		}
	}
	readVersion := c.getReadVersion()
	storedMsgs, _, err := c.messageCrud.ReadBatch(uncachedIds)
	if err != nil {
		return nil, nil, err
	}
	for _, msg := range storedMsgs {
		c.cache.Add(c.stringKey(msg), msg, readVersion)
	}
	orderedResults := make([]proto.Message, 0, len(cachedMsgs)+len(storedMsgs))
	missingIndices := make([]int, 0, len(ids)-len(cachedMsgs)-len(storedMsgs))
	cachedIndex := 0
	storedIndex := 0
	for idx := range ids {
		if cachedIndex < len(cachedMsgs) && c.stringKey(cachedMsgs[cachedIndex]) == ids[idx] {
			orderedResults = append(orderedResults, cachedMsgs[cachedIndex])
			cachedIndex++
		} else if storedIndex < len(storedMsgs) && c.stringKey(storedMsgs[storedIndex]) == ids[idx] {
			orderedResults = append(orderedResults, storedMsgs[storedIndex])
			storedIndex++
		} else {
			missingIndices = append(missingIndices, idx)
		}
	}
	return orderedResults, missingIndices, nil
}

func (c *cachedMessageCrudImpl) Update(msg proto.Message) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.Update(msg)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	c.cache.Add(c.stringKey(msg), msg, writeVersion)
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) UpdateBatch(msgs []proto.Message) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.UpdateBatch(msgs)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	for _, key := range msgs {
		c.cache.Add(c.stringKey(key), key, writeVersion)
	}
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) Upsert(msg proto.Message) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.Upsert(msg)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	c.cache.Add(c.stringKey(msg), msg, writeVersion)
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) UpsertBatch(msgs []proto.Message) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.UpsertBatch(msgs)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	for _, key := range msgs {
		c.cache.Add(c.stringKey(key), key, writeVersion)
	}
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) Delete(id string) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.Delete(id)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	c.cache.Remove(id, writeVersion)
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) DeleteBatch(ids []string) (uint64, uint64, error) {
	writeVersion, attempts, err := c.messageCrud.DeleteBatch(ids)
	defer c.addReadVersion(attempts)
	if err != nil {
		return writeVersion, attempts, err
	}
	for _, id := range ids {
		c.cache.Remove(id, writeVersion)
	}
	return writeVersion, attempts, nil
}

func (c *cachedMessageCrudImpl) Count() (int, error) {
	return c.messageCrud.Count()
}

func (c *cachedMessageCrudImpl) Create(msg proto.Message) error {
	return c.messageCrud.Create(msg)
}

func (c *cachedMessageCrudImpl) CreateBatch(msgs []proto.Message) error {
	return c.messageCrud.CreateBatch(msgs)
}

func (c *cachedMessageCrudImpl) ReadAll() ([]proto.Message, error) {
	return c.messageCrud.ReadAll()
}

func (c *cachedMessageCrudImpl) KeyFunc(message proto.Message) []byte {
	return c.messageCrud.KeyFunc(message)
}
