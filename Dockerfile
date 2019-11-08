FROM gcr.io/jenkinsxio/builder-go:2.0.973-309

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]