all: dumper server

dumper:
	javac -cp minecraft-0.30.jar LevelDumper.java

server:
	go build

clean:
	rm -f mcmuseum
	rm -f LevelDumper.class

.PHONY: clean
