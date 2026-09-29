# Nezumi Music Tracker

_Nezumi_ is a music (keygen) tracker that works totally from a TUI (terminal user interface).

CURRENTLY EXPERIMENTAL! Expect breaking changes!

## How does it work?

Nezumi uses `libopenmpt` under the hood for reading the modules, which is a primarily C++ library, that is likely on your system already, but it does provide some regular C code as well.

Under `libopenmpt/`, you can find a copy of `libopenmpt.h` (for development purposes, may be removed at some point), and the CGO bindings.

## Usage

```sh
nezumi "path-to-module"
```

It supports the common tracker module formats, such as MOD (Amiga), IT (Impulse Tracker), XM (ProTracker), S3M (SchismTracker 3), among many others.

Check `nezumi -help` for more commands.

## Development

Requirements:

- Go 1.26+
- GCC or Clang (for CGO bindings)
- `libopenmpt`

Recommended, but not mandatory:

- just
- valgrind

There's also find a `flake.nix` for a contained development environment with the needed tools.

### Building

```sh
just build
# or if not using just:
go build ./cmd/nezumi
```

Remember to **not** disable CGO bindings when building because `libopenmpt` is not statically linked (at least yet)!

## To-do?

- [ ] finish playback (allow rewind, next, stop)
- [ ] show more relevant metadata (message, samples, etc.)
- [ ] make the visualization modes
- [ ] maybe an edit mode? Would be good, but probably the hardest one
- [x] ~~move main.go to `cmd/nezumi` so it can support more commands~~
  - ~~maybe also bring cobra or uv for handling CLI~~
- [ ] check whether statically linking `libopenmpt` is viable, so it could be a self-contained (albeit larger) binary. Could be useful for people on other OSes
