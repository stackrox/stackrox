
check: imports
	go test ../handler

imports: generate
	test -z "$(shell goimports -l .)" || ( goimports -d .  ; exit 1)

generate:
	go run ./gen

clean:
	rm -f generated.go
