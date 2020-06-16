FROM gcr.io/jenkinsxio/builder-go:2.1.68-699

COPY ./build/lighthouse-githubapp-linux-amd64 /lighthouse

EXPOSE 8080

CMD ["/lighthouse"]