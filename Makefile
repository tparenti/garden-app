.PHONY: build run tidy clean

build:
	go build -o garden.exe .

serve:
	go run . serve --port 8080

run:
	go run . $(ARGS)

tidy:
	go mod tidy

clean:
	rm -f garden.exe
