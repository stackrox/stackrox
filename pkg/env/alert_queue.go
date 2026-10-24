package env

// AlertQueueMaxShardSize caps the number of alert messages buffered in a single shard of
// Central's per-connection alert worker queue. Sensor can emit runtime alerts faster than
// the pipeline drains them, and all of a single deployment's alerts hash to one shard, so
// without a cap a noisy deployment grows its shard unbounded and balloons Central's heap
// under load. The cap is per shard (not a shared total) so that one busy shard cannot
// evict unique messages queued for other, quieter deployments on different shards. With
// the current 16 alert shards the default bounds the queue at ~160k buffered messages.
// 0 disables the cap (unbounded, the legacy behavior).
// See central/sensor/service/connection/worker_queue.go.
var AlertQueueMaxShardSize = RegisterIntegerSetting("ROX_ALERT_QUEUE_MAX_SHARD_SIZE", 10000).WithMinimum(0)
