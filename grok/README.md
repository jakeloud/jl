## Build steps

```
docker build --tag=jl --file=build.Dockerfile .
```

```
docker create --name jlc jl
```

```
docker cp jlc:/app/jl jl
```

```
docker rm jlc
```

## Setup environment

```
docker build --tag jakeloud-env --file=env.Dockerfile .
```

## Test steps

```
docker build --tag jlt --file=test.Dockerfile .
```

```
docker run --name=jakeloud-test -p 80:80 -p 443:443 -p 666:666 jlt
```
