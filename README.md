## Build steps

```
docker build --tag=jl --file=build.Dockerfile .
docker create --name jlc jl
docker cp jlc:/app/jl jl
docker rm jlc
```

## Test steps

```
docker build --tag jlt --file=test.Dockerfile .
```

```
docker run --name=jakeloud-test -p 80:80 -p 443:443 -p 666:666 -it jlt
```


## Almost All in one
```
docker build --tag=jl --file=build.Dockerfile .
docker create --name jlc jl
docker cp jlc:/app/jl jl
docker rm jlc
docker build --tag jlt --file=test.Dockerfile .
docker run --name=jakeloud-test -p 80:80 -p 443:443 -p 666:666 -it jlt
```
