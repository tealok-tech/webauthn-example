# webauthn-example

This project shows a simple example of how to write both a frontend (JavaScript) and backend (Go) system for using [Webauthn](https://webauthn.io/).

The backend relies on [the go webauthn library](github.com/go-webauthn/webauthn).

The frontend just uses vanilla JavaScript with no libraries.

## Building

```
go build
```

This will produce `webauthn-example` in the current directory

## Running

```
./webauthn-example
```

This will run the process on port 8080 of the local host, binding to all available addresses. You can view the UI in a web browser at http://localhost:8080.
