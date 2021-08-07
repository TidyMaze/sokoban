.PHONY: build build-profile run-profile profile-all display-cpu-profile display-heap-profile
build:
	go build -o out/sokoban

build-profile:
	go build -o out/sokoban-profile

run-profile: build-profile
	./out/sokoban-profile profile

display-cpu-profile:
	pprof -http=:8080 out/out.prof

display-heap-profile:
	pprof -http=:8080 out/mem.prof

live-heap-profile:
	pprof -http=:8080 http://localhost:6060/debug/pprof/heap

profile-all: run-profile live-heap-profile