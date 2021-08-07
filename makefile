.PHONY: build, profile
build:
	go build -o out/sokoban

profile:
	go build -o out/sokoban-profile

run-profile:
	./out/sokoban-profile