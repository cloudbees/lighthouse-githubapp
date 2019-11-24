FROM gcr.io/jenkinsxio/builder-go:2.0.976-312

ARG VERSION

RUN echo "building image version $VERSION"

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

COPY source-context.json /source-context.json

RUN go get -u cloud.google.com/go/cmd/go-cloud-debug-agent \
    go-cloud-debug-agent -sourcecontext=/source-context.json -appmodule=lighthouse-githubapp \
    -appversion=$VERSION -- /lighthouse
    
EXPOSE 8080

CMD ["/lighthouse"]