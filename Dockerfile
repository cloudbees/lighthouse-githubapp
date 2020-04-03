FROM gcr.io/jenkinsxio/builder-go:2.0.1265-605

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]