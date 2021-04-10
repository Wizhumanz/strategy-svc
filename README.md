# Strategy Service, Anastasia Trading Bot

Strategy service to call orders for Anastasia.

## Theory

Process for executing trade actions
1. TV webhook calls `/webhook` route in `api-gateway`
2. api-gateway adds new trade with key `<aggregateID>:<userID>:<botID>` into `webhookTrades` stream
3. `strategy-svc` starts saga for each msg in `webhookTrades` stream
4. Saga started by `strategy-svc` listened by `analytics-svc` and `order-svc`

## Local Dev

```
chmod +x run.sh
./run.sh
```

Inside `run.sh`:
```
export PORT=8000

# build/ directory ignored by git
go build -o build/api .

build/api
```

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
