dev module:
  go run ./cmd/nezumi {{ module }}

build:
  go build -o build/ ./cmd/nezumi

test:
  go test ./...

lint:
  golangci-lint fmt
  golangci-lint run ./... --fix

valgrind:
  go build -tags valgrind -o build/nezumi-memcheck ./cmd/nezumi
  valgrind --tool=memcheck --leak-check=full --show-leak-kinds=all ./build/nezumi-memcheck
