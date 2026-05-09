# sample-grpc process

A minimal backend process for the `process-grpc` channel.

Current transport is JSON-RPC over TCP (compatible with the backend `process-grpc` executor).

Run manually:

```bash
go run .
```

It listens on `127.0.0.1:19104` and handles `PluginGateway.Handle`.
