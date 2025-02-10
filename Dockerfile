FROM golang:1.23.3 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /workspace/frpc-controller .

FROM snowdreamtech/frpc

WORKDIR /workspace

COPY --from=builder /workspace/frpc-controller /workspace/frpc-controller

ENTRYPOINT [ "./frpc-controller" ]