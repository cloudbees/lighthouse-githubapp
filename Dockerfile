FROM gcr.io/jenkinsxio/builder-go:2.0.1216-556

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]