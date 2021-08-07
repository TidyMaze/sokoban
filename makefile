.PHONY: build build-profile run-profile profile-all display-cpu-profile display-heap-profile
build:
	go build -o out/sokoban

build-profile:
	go build -o out/sokoban-profile

run-profile:
	./out/sokoban-profile profile

display-cpu-profile:
	pprof -http=:8080 out/out.prof

display-heap-profile:
	pprof -http=:8080 out/mem.prof

profile-all: build-profile run-profile display-cpu-profile