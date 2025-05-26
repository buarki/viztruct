![Go Tests](https://github.com/buarki/viztruct/actions/workflows/tests.yml/badge.svg) [![Vercel Deploy](https://deploy-badge.vercel.app/vercel/viztruct)](https://viztruct.vercel.app/) [![tag and release](https://github.com/buarki/viztruct/actions/workflows/release.yml/badge.svg)](https://github.com/buarki/viztruct/actions/workflows/release.yml)


# viztruct


![Image](./docs/demo.gif)
SVG visualization:

![Image](./docs/demo.png)

## CLI installation

### Go install

```sh
go install github.com/buarki/viztruct/cmd/viztruct@latest
```

### Download binaries

```sh
ARCH="arm64" # or amd64
OS="darwin" # or linux

# get latest tag using GitHub API
VERSION=$(curl -s "https://api.github.com/repos/buarki/viztruct/releases/latest" | jq -r .tag_name)

# download binary
BINARY_URL="https://github.com/buarki/viztruct/releases/download/$VERSION/viztruct-$OS-$ARCH"
curl -L "$BINARY_URL" -o viztruct

# install
chmod +x viztruct
sudo mv viztruct /usr/local/bin/

# verify
viztruct --version
```

### Build it locally

Build the CLI:
```sh
git clone git@github.com:buarki/viztruct.git
cd viztruct
make build-cli
sudo mv viztruct /usr/local/bin/
```

## Usage:
### Analyze a struct from command line

```sh
viztruct --struct 'type MyStruct struct { A int8; B int32 }'
```

### Analyze structs from a file

```sh
viztruct --path ./internal/samples/dumb_service.go --format=txt
```

### Get JSON output
```sh
viztruct --format json --struct 'type MyStruct struct { A int8; B int32 }'
```

### Generate SVG visualization
```sh
viztruct --svg --struct 'type MyStruct struct { A int8; B int32 }'
```

### Analyze and define the optimization strategy
```sh
viztruct --path ./internal/samples/dumb_service.go --format=txt --strategies=alignment,size,group,greedy
```

### Show help
```sh
viztruct --help
```

The tool will print the struct layout analysis to stdout. Use the `--svg` flag to generate an SVG visualization.

## Website

If you want to use from browser just visit the [deployed webapp](https://viztruct.vercel.app). You can paste/type your struct in the text input area and get a full padding analysis.

## Limitations
For now it is not able to handle directory paths as input. It's in the pipeline.
