package timeutil

import "time"

// Max is the maximum value that can be represented in a `time.Time` object.
var Max = time.Unix(1<<63-62135596801, 999999999)
