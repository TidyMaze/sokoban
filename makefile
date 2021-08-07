.PHONY: build, profile, profile-all
build: app.go
	go build -o out/sokoban

build-profile: app.go
	go build -o out/sokoban-profile

run-profile: ./out/sokoban-profile
	./out/sokoban-profile

display-profile:
	pprof -http=:8080 out/sokoban-profile out/out.prof

profile-all: build-profile run-profile display-profile