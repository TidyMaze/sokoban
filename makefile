.PHONY: build build-profile run-profile profile-all display-profile
build: app.go
	go build -o out/sokoban

build-profile: app.go
	go build -o out/sokoban-profile

run-profile: out/sokoban-profile
	./out/sokoban-profile

display-profile: out/out.prof
	pprof -http=:8080 out/out.prof

profile-all: build-profile run-profile display-profile