export PATH := /usr/local/go/bin:$(PATH)

dm-cli: force
	go build -o dm-cli

dm-cli.exe: force
	GOOS=windows GOARCH=amd64 go build -o dm-cli.exe

graph:
	gnuplot -p graph.gnuplot && xdg-open telemetry.png

force:
