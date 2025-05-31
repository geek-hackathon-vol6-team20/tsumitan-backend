# Project hello

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development purpose. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Build the application
```bash
make all
# or
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

Live reload the application:
```bash
make watch
```

Clean up binary from the last build:
```bash
make clean
```
