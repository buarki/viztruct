![Go Tests](https://github.com/buarki/viztruct/actions/workflows/tests.yml/badge.svg) [![Vercel Deploy](https://deploy-badge.vercel.app/vercel/viztruct)](https://viztruct.vercel.app/) [![tag and release](https://github.com/buarki/viztruct/actions/workflows/release.yml/badge.svg)](https://github.com/buarki/viztruct/actions/workflows/release.yml)


# viztruct


![Image](./docs/demo.gif)
SVG visualization:

![Image](./docs/demo.png)

## CLI installation

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
```sh
# Analyze a struct from command line
viztruct --struct 'type MyStruct struct { A int8; B int32 }'

# Analyze structs from a file
viztruct --file ./samples/bad-layout.txt

# Get JSON output
viztruct --format json --struct 'type MyStruct struct { A int8; B int32 }'

# Generate SVG visualization
viztruct --svg --struct 'type MyStruct struct { A int8; B int32 }'

# Show help
viztruct --help
```

The tool will print the struct layout analysis to stdout. Use the `--svg` flag to generate an SVG visualization.

## Website

If you want to use from browser just visit the [deployed webapp](https://viztruct.vercel.app). You can paste/type your struct in the text input area and get a full padding analysis.



