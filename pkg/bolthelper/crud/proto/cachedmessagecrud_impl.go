package proto

import (
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/expiringcache"
)

type cachedMessageCrudImpl struct {
	messageCrud MessageCrud

	metricType string
	metricFunc func(string, string)

	cacheLock sync.Mutex
	cache     expiringcache.Cache
}

func (c *cachedMessageCrudImpl) stringKey(msg proto.Message) string {
	return string(c.KeyFunc(msg))
}

func (c *cachedMessageCrudImpl) Read(id string) (proto.Message, error) {
	if cached := c.cache.Get(id); cached != nil {
		c.metricFunc("hit", c.metricType)
		return proto.Clone(cached.(proto.Message)), nil
	}
	c.metricFunc("miss", c.metricType)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	msg, err := c.messageCrud.Read(id)
	if msg != nil {
		c.cache.Add(id, msg)
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
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	storedMsgs, _, err := c.messageCrud.ReadBatch(uncachedIds)
	if err != nil {
		return nil, nil, err
	}
	for _, msg := range storedMsgs {
		c.cache.Add(c.stringKey(msg), msg)
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

func (c *cachedMessageCrudImpl) Update(msg proto.Message) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.Update(msg); err != nil {
		return err
	}
	c.cache.Add(c.stringKey(msg), msg)
	return nil
}

func (c *cachedMessageCrudImpl) UpdateBatch(msgs []proto.Message) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.UpdateBatch(msgs); err != nil {
		return err
	}
	for _, key := range msgs {
		c.cache.Add(c.stringKey(key), key)
	}
	return nil
}

func (c *cachedMessageCrudImpl) Upsert(msg proto.Message) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.Upsert(msg); err != nil {
		return err
	}
	c.cache.Add(c.stringKey(msg), msg)
	return nil
}

func (c *cachedMessageCrudImpl) UpsertBatch(msgs []proto.Message) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.UpsertBatch(msgs); err != nil {
		return err
	}
	for _, key := range msgs {
		c.cache.Add(c.stringKey(key), key)
	}
	return nil
}

func (c *cachedMessageCrudImpl) Delete(id string) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.Delete(id); err != nil {
		return err
	}
	c.cache.Remove(id)
	return nil
}

func (c *cachedMessageCrudImpl) DeleteBatch(ids []string) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if err := c.messageCrud.DeleteBatch(ids); err != nil {
		return err
	}
	for _, id := range ids {
		c.cache.Remove(id)
	}
	return nil
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
