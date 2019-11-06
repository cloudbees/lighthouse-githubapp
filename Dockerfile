FROM gcr.io/jenkinsxio/builder-go:2.0.961-300

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]