FROM golang:1.19.3-alpine3.16 AS build

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v

FROM alpine:3.16 as run

WORKDIR /app/
COPY requests.json .
COPY --from=build /app/fhir-bomber .
COPY --from=build /app/app.yml .

EXPOSE 8081

ENTRYPOINT ["/app/fhir-bomber"]
