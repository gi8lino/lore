# syntax=docker/dockerfile:1.27

FROM node:24-alpine AS web
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json ./
COPY scripts/web ./scripts/web
COPY web/src ./web/src

RUN ./scripts/web/build.sh


FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web /src/web/dist ./web/dist

RUN go generate ./internal/icons \
  && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/lore ./cmd


FROM alpine:3.24

RUN apk add --no-cache weasyprint font-noto \
  && addgroup -S lore \
  && adduser -S -G lore lore

COPY --from=build /out/lore /usr/local/bin/lore

USER lore

EXPOSE 8080
ENTRYPOINT ["lore"]
CMD ["serve"]
