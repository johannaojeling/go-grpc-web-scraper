# Go gRPC Web Scraper

This project contains a gRPC service with a server-side streaming RPC that scrapes URLs using Colly. The server can be
deployed to Cloud Run. An example client application can be found in [cmd/client/main.go](cmd/client/main.go), where
messages received from the stream are written to a json file.

## Pre-requisites

- Go version 1.18
- Protoc version 3
- Protoc Protobuf and gPRC Go plugins
- Gcloud SDK

## Development

### Setup

Install Go dependencies

```bash
go mod download
```

Regenerate Protobuf and gRPC code if needed

```bash
protoc --go_out=. \
--go-grpc_out=. \
api/**/*.proto
```

### Running the application

Set environment variables

| Variable | Description                                        |
|----------|----------------------------------------------------|
| PORT     | Port for server                                    |

Run application

```bash
go run cmd/server/main.go
```

### Calling the service

Run example client application

```bash
go run cmd/client/main.go --port=${PORT} --withInsecure
```

## Deployment

### Deploying to Cloud Run

Set environment variables

| Variable           | Description                                 |
|--------------------|---------------------------------------------|
| PROJECT            | GCP project                                 |
| REGION             | Region                                      |
| CLOUD_BUILD_BUCKET | Bucket used for Cloud Build                 |
| SA_EMAIL           | Email of service account used for Cloud Run |
| IMAGE              | Image tag                                   |
| SERVICE            | Service name                                |

Build image with Cloud Build

```bash
gcloud builds submit . \
--project=${PROJECT} \
--config=./cloudbuild.yaml \
--gcs-source-staging-dir=gs://${CLOUD_BUILD_BUCKET}/staging \
--substitutions=_PROJECT=${PROJECT},_LOGS_BUCKET=${CLOUD_BUILD_BUCKET},_IMAGE=${IMAGE}
```

Deploy to Cloud Run

```bash
gcloud run deploy ${SERVICE} \
--project=${PROJECT} \
--region=${REGION} \
--image=eu.gcr.io/${PROJECT}/${IMAGE} \
--service-account=${CLOUD_RUN_SA} \
--platform=managed \
--use-http2 \
--no-allow-unauthenticated
```

### Calling the service

Set host and token

```bash
export URL=$(gcloud run services describe ${SERVICE} \
--project=${PROJECT} \
--region=${REGION} \
--format='value(status.url)')

export HOST=${URL#https://}

export TOKEN=$(gcloud auth print-identity-token)
```

Run example client application

```bash
go run cmd/client/main.go --host=${HOST} --port=443 --token=${TOKEN}
```
