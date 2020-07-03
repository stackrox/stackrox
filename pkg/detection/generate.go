package detection

//go:generate genny -in=../maputil/maputil.go -out=map_gen.go -pkg detection gen "KeyType=string ValueType=CompiledPolicy"
