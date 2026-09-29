build:
  go build -o build/ ./cmd/nezumi

valgrind:
  go build -tags valgrind -o build/nezumi-memcheck ./cmd/nezumi
  valgrind --tool=memcheck --leak-check=full --show-leak-kinds=all ./build/nezumi-memcheck
