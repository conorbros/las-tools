# Lastools

## Running the project

### Docker

If you don't have Go installed, running with docker is probably the easiest way to run it locally. There is a script in the `scripts` folder called `run_docker.sh`. That is all that is needed to run it with Docker. You can then navigation to `localhost:8080` in your browser to use the application.

### Go

If you want to run the application locally with Go you must first [install Go](https://golang.org/doc/install). After that is complete, clone the repo to your `goroot` and run `go mod download` in the project directory. Then `go run main.go` which will start the application and it can be viewed at `localhost:8080` in the browser.

![image](web/static/assets/GOPHER_ROCKS.png)