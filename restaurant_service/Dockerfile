FROM golang:1.27

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY restaurant_service ./restaurant_service

RUN CGO_ENABLED=0 go build -o restaurant-service ./restaurant_service

EXPOSE 8081

CMD ["./restaurant-service"]