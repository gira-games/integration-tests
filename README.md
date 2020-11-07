## Gira - integration tests

This are the integration tests of the Gira project. They are self-contained. They can run stand-alone, and will be able to download and start everything they need.

### How to run
```
$ go test ./...
```
This will spin up a PostgreSQL DB, and an API server, and will use them to run the scenarious defined in the test files.

You can also:
```
$ go test ./... -api-image my-repo/my-gira-api-image -api-version v1.0.0 -postgres-version 12
```
To use custom version and images.
### License
This work is licensed under MIT license. For more info see [LICENSE.md](LICENSE.md)
