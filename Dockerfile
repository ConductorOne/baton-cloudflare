FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-cloudflare"]
COPY baton-cloudflare /