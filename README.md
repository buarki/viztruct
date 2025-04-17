![Go Tests](https://github.com/buarki/viztruct/actions/workflows/tests.yml/badge.svg) [![Vercel Deploy](https://deploy-badge.vercel.app/vercel/viztruct)](https://viztruct.vercel.app/)


# viztruct

## CLI

- build it:

```sh
make build-cli
```

- it accepts the structs to analyse via stdin or via pipe:

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

- it will give you a padding summary and also generate an svg image of the input and optimized struct.

## Website

If you want to use from browser just visit the [deployed webapp](https://viztruct.vercel.app). You can paste/type your struct in the text input area and get a full padding analysis.



