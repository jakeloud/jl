# Build steps

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
