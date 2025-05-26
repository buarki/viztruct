# viztruct pipeline plugin

## How It Works

- It collects the changed files between two commits;
- Then, it perform an analisy at these changed files;
- The output is a summary of the struct layout analyse;

Ex: a simple usage:

```sh
docker run --rm -v $(pwd):/repo ci-plugin --repo .  --verbose
```

Above example it consider current commit (HEAD) as the `target` commit and will use previous commit as the `origin`;

You can also manualy set the `to` and `from`:
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
