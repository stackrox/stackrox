package postgres

//go:generate pg-table-bindings-wrapper --type=WatchedImage --table=watchedimages --key-func GetName()
