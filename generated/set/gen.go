package set

//go:generate genny -in=../../pkg/set/generic.go -out=v1-search-cats-set.go -imp=github.com/stackrox/rox/generated/api/v1 gen "KeyType=v1.SearchCategory"
//go:generate genny -in=../../pkg/set/generic.go -out=upgrade-progress-state-set.go -imp=github.com/stackrox/rox/generated/storage gen "KeyType=storage.UpgradeProgress_UpgradeState"
