FROM gcr.io/jenkinsxio/builder-go:2.0.903-251
#RUN apk add --update --no-cache ca-certificates git

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]