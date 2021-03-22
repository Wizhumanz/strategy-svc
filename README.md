# Strategy Service, Anastasia Trading Bot

Strategy service to call orders for Anastasia.

## Local Dev

Serve command: `PORT=8000 go run main.go`

`PORT` env var must be passed for local dev. This env var is passed by default when deployed to Cloud Run.

### [GCP Datastore testing](https://cloud.google.com/datastore/docs/reference/libraries#client-libraries-install-go):

1. Must authenticate: `export GOOGLE_APPLICATION_CREDENTIALS="/path/to/auth/my-key.json"` in current shell session.

OR local testing with emulator:

1. Install local emulator: `gcloud components install cloud-datastore-emulator`.
2. [Run local emulator](https://cloud.google.com/datastore/docs/tools/datastore-emulator): `gcloud beta emulators datastore start`.
3. Set env vars so that local app connects to emulated DB instead of prod: `$(gcloud beta emulators datastore env-init)`.

### Docker

```
cd api
docker build -t <img-name> .
docker run -e AUTH=password -e PORT=8000 --name <container-name> -p 8000:8000 <img-name>
```
