# Development

A Docker Compose file is provided. In order for webspaced to be able to connect to LXD, you'll need to provide a client
certificate and key (already trusted by LXD) in the `certs/` directory. You should also run a `kubectl proxy` so that
webspaced can talk to your Kubernetes cluster from your machine. _Ensure that kubectl is configured to connect to a
development cluster so that you don't interfere with production!_

This repo makes use of the `build.yaml`, `release.yaml`, `generate.yaml`, `charts.yaml` and `docs.yaml` GitHub Actions
workflows as described in the [IAM documentation](../../../iam/development/#github-actions).

## Maintenance

- To make a new release, push a tag of the semver form supported by the `release.yaml` workflow
- To upgrade Go, make sure to update the `Dockerfile` base image as well as the version in `go.mod`
- When upgrading Go dependencies, keep in mind that webspaced needs to depend on Netsoc's fork of Traefik due to changes
  to custom resources
