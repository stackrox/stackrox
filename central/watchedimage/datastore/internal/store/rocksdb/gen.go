package rocksdb

//go:generate rocksdb-bindings-wrapper --type=WatchedImage --bucket=watchedimages --key-func GetName() --cache
