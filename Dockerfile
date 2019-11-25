FROM gcr.io/jenkinsxio/builder-go:2.0.1014-345

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]