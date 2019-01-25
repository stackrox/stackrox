
check: imports
	go test ../handler

imports: generate
	test -z "$(shell goimports -l .)" || ( goimports -d .  ; exit 1)

generate:
	go generate .

clean:
	rm -f generated.go
