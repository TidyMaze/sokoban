.PHONY: build build-profile run-profile profile-all display-profile
build:
	go build -o out/sokoban

build-profile:
	go build -o out/sokoban-profile

run-profile:
	./out/sokoban-profile profile

display-profile:
	pprof -http=:8080 out/out.prof

profile-all: build-profile run-profile display-profile