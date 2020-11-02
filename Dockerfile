FROM golang:latest

# Set environment variables.
RUN go get github.com/tools/godep

# Add application code
ADD . /go/src/github.com/conorbros/conorb-dev

# Set working dir to app code dir
WORKDIR /go/src/github.com/conorbros/conorb-dev

# Download dependencies and build the application
RUN go mod download
RUN go build

# Expose port 80 and set as PORT environment variable for Go
EXPOSE 80
ENV PORT 80
ENV SPOTIFY_REDIRECT_URL "https://conorb.dev/playlist"

# Run the application
CMD ["./conorb-dev"]