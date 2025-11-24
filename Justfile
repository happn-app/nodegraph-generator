buildarch os arch:
  mkdir -p dist
  CGO_ENABLED=0 GOOS={{ os }} GOARCH={{ arch }} go build -ldflags="-s -w" -o ./dist/nodegraph-generator-{{ os }}-{{ arch }}

build:
  just buildarch linux amd64
  just buildarch linux arm64
  just buildarch darwin amd64
  just buildarch darwin arm64

grafana:
  docker compose up -d && open http://localhost:3000
