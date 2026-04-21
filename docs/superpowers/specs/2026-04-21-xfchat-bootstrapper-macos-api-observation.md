# Xfchat Bootstrapper macOS API Observation Notes

**Date:** 2026-04-21

## Recorded Operations

### Create App

- Method: `POST`
- Path: `/api/apps`
- Query params: none
- Body fields: `name`
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app`; `Accept: application/json`; `Content-Type: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: `data.app_id`, `data.app_url`
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode failure reported as a response decode error
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

### Get Credentials

- Method: `GET`
- Path: `/api/apps/{app_id}/credentials`
- Query params: none
- Body fields: none
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app/{app_id}/baseinfo`; `Accept: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: `data.app_id`, `data.app_secret`, `data.app_url`
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode failure reported as a response decode error; missing `data.app_secret` is treated as an error
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

### Ensure Redirect URL

- Method: `PUT`
- Path: `/api/apps/{app_id}/safe`
- Query params: none
- Body fields: `app_id`, `callback_url`
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app/{app_id}/safe`; `Accept: application/json`; `Content-Type: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: none required by the current client
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode is not required for this call
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

### Ensure Scopes

- Method: `PUT`
- Path: `/api/apps/{app_id}/auth`
- Query params: none
- Body fields: `app_id`, `scopes`
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app/{app_id}/auth`; `Accept: application/json`; `Content-Type: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: none required by the current client
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode is not required for this call
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

### Create Version

- Method: `POST`
- Path: `/api/apps/{app_id}/version`
- Query params: none
- Body fields: `app_id`
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app/{app_id}/version`; `Accept: application/json`; `Content-Type: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: none required by the current client
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode is not required for this call
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

### Publish Version

- Method: `POST`
- Path: `/api/apps/{app_id}/version/publish`
- Query params: none
- Body fields: `app_id`
- Required cookies/headers: session cookies from the logged-in browser profile; `Origin`; `Referer: https://open.xfchat.iflytek.com/app/{app_id}/version`; `Accept: application/json`; `Content-Type: application/json`; `X-XSRF-TOKEN` when available
- Success response fields: none required by the current client
- Failure shape: non-2xx HTTP status with response text included in the error; JSON decode is not required for this call
- Note: this contract is implementation-inferred from `internal/platformapi/client.go` and must be confirmed by real macOS capture

## Required Request Context

- cookies from logged-in browser profile
- CSRF or anti-forgery header
- origin and referer matching `open.xfchat.iflytek.com`
- request-scoped cookie selection based on the request URL, not just the referer
