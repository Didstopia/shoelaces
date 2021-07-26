FROM golang:1.16-alpine AS build

WORKDIR /shoelaces
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"' -o /tmp/shoelaces . && \
    printf "---\nnetworkMaps:\n" > /tmp/mappings.yaml



# Final container has basically nothing in it but the executable
# FROM scratch
FROM golang:1.16-alpine
COPY --from=build /tmp/shoelaces /shoelaces

RUN mkdir -p /shoelaces_default/{data,web} /data /web

COPY --from=build /tmp/mappings.yaml /shoelaces_default/data/mappings.yaml
COPY --from=build /shoelaces/web /shoelaces_default/web

COPY docker_entrypoint.sh /entrypoint
RUN chmod +x /entrypoint

#ENV BIND_ADDR=0.0.0.0:8081
ENV PUID 1000
ENV PGID 100
EXPOSE 80

VOLUME [ "/data", "/web" ]

# ENTRYPOINT ["/shoelaces", "-data-dir", "/data", "-static-dir", "/web"]
# CMD []

ENTRYPOINT ["/entrypoint"]
CMD ["/shoelaces", "-bind-addr", "0.0.0.0:80", "-data-dir", "/data/", "-static-dir", "/web/", "-template-extension", ".slc", "-mappings-file", "mappings.yaml", "-debug", "true"]

# ENTRYPOINT ["/entrypoint", "/shoelaces", "-data-dir", "/data", "-static-dir", "/web"]
