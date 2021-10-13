package postgres

//go:generate pg-bindings-wrapper --type=WatchedImage --table=watchedimages --key-func GetName()
