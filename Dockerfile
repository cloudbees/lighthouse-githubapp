FROM gcr.io/jenkinsxio/builder-go:2.0.1049-382

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]