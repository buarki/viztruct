![Go Tests](https://github.com/buarki/viztruct/actions/workflows/tests.yml/badge.svg) [![Vercel Deploy](https://deploy-badge.vercel.app/vercel/viztruct)](https://viztruct.vercel.app/)


# viztruct

## Cli

```sh
make build-cli
```

then use it:

```sh
./viztruct 'type ComplexStruct struct { Name     string; ID       uint64; Active   bool; Count    int32; Flags    byte; Value    float64; Reserved bool }'
```

or 

```sh
echo 'type MyStruct struct {
  A int8
  B int32
}' | ./viztruct
```

