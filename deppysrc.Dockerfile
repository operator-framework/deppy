FROM golang:1.17.6-alpine as Build

COPY . . 

RUN GOPATH= CGO_ENABLED=0 go build -o /build/catalog_source_controller internal/entitysource/adapter/catalogsource/cmd/cmd.go


FROM scratch

COPY --from=Build /build/catalog_source_controller catalog_source_controller

USER 1001
EXPOSE 50052

CMD ["./catalog_source_controller"]
