  
FROM golang:1.14-alpine AS build
WORKDIR /src/
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app *.go

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /src/
COPY --from=build /src/ .
CMD ["/src/app"] 