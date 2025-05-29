# viztruct pipeline plugin

## How It Works

- It collects the changed files between two commits
- Then, it performs an analysis on these changed files
- The output is a summary of the struct layout analysis

A simple usage example:

```sh
docker run --rm -v $(pwd):/repo ci-plugin --repo .  --verbose
```

The above example considers the current commit (HEAD) as the `target` commit and will use the previous commit as the `origin`.

You can also manually set the `to` and `from`:
```sh
docker run --rm -v $(pwd):/repo ci-plugin --repo .  --verbose --from=SOME_COMMIT_HERE --to=ANOTHER_COMMIT_HERE
```


## Developing

```sh
make build-plugin
```

## Image usage

### Build
```sh
docker build -f cmd/ci-plugin/Dockerfile -t ci-plugin .
```

### Running

```sh
docker run --rm -v $(pwd):/repo ci-plugin --repo .  --verbose
```
