package rocksdb

//go:generate rocksdb-bindings-wrapper --type=WatchedImage --bucket=watchedimages --key-func GetName() --cache --migrate-seq 57 --migrate-to watched_images
