![Go Tests](https://github.com/buarki/viztruct/actions/workflows/tests.yml/badge.svg) [![Vercel Deploy](https://deploy-badge.vercel.app/vercel/viztruct)](https://viztruct.vercel.app/)


# viztruct

## CLI

Build the CLI:
```sh
make build-cli
```

Usage:
```sh
# Analyze a struct from command line
./viztruct --struct 'type MyStruct struct { A int8; B int32 }'

# Analyze structs from a file
./viztruct --file structs.go

# Get JSON output
./viztruct --format json --struct 'type MyStruct struct { A int8; B int32 }'

# Generate SVG visualization
./viztruct --svg --struct 'type MyStruct struct { A int8; B int32 }'

# Show help
./viztruct --help
```

The tool will print the struct layout analysis to stdout. Use the `--svg` flag to generate an SVG visualization.

## Website

If you want to use from browser just visit the [deployed webapp](https://viztruct.vercel.app). You can paste/type your struct in the text input area and get a full padding analysis.



