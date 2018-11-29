
check: generate
	go test ../handler

generate:
	go generate . && goimports -w .

clean:
	rm -f generated.go
