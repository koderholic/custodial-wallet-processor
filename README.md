# wallet-adapter
The wallet adapter service handles the wallet functionalities for users crypto asset and it's interactions with other crypto gateway services.

Setup :

Ensure you have Golang installed and your GOPATH / GOROOT are set correctly on your computer (see https://golang.org/doc/install for installations)

clone repository

To get a quick feel and flow of the service, navigate to the project root : 

Run "go install" to install packages and dependencies

Run "go build" to compile into executable. You can also run any of the bash script at the root, to cross-compile and executable will be saved to "build" folder

Double-click on executable, the golang web server will start and listen for connections on the configured PORT

API Specifications can be viewed on swagger here => localhost:{PORT}/swagger/

Developer Setup :

- Create a config.yaml file from the config-default.yaml file and replace necessary fields i.e AUTHENTICATION_SERVICE_SERVICE_ID, AUTHENTICATION_SERVICE_TOKEN, SECURITY_BUNDLE_PUBLICKEY for authenticating request
- Build the service by running "go build"
- Run the built executable "./walletAdapter"

## Dependency

It needs a mysql db. Db name and db server location can be configured in the config.yml file, see sample in config-default.yaml

## Running Tests

The quickest way to run test is by using go run:
`bash
    $ go test -v ./tests
`