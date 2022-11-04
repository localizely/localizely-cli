FROM golang:1.16 AS build

WORKDIR /build

COPY go.mod .
COPY go.sum .
COPY main.go .
COPY cmd cmd
RUN go mod download

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -o /bin/localizely-cli

FROM scratch
COPY --from=build /bin/localizely-cli /bin/localizely-cli
ENTRYPOINT [ "/bin/localizely-cli" ] 
