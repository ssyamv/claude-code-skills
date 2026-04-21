# Xfchat Bootstrapper Auto Callback And App Design

## Goal

Make bootstrap setup more automatic when the default OAuth callback port is unavailable and when the expected `lark_cli` application does not already exist.

## Runtime Callback URL

The bootstrapper should prefer `127.0.0.1:8080` for the local OAuth callback server. If that bind fails, it should bind `127.0.0.1:0` and use the actual listener port returned by the OS.

The actual callback URL is the runtime source of truth for:

- the application redirect URL configured in the platform
- the generated OAuth authorization URL
- the OAuth callback waiter

The static config callback URL remains a preferred default, not a guarantee.

## Application Setup

The platform client should first search for an existing app named `lark_cli`. If it exists, setup should reuse it. If it does not exist, setup should create the app and continue with the same configuration flow.

After an app is available, setup should:

- ensure the actual callback URL is configured
- ensure required scopes are available
- create a version when needed by the current API flow
- publish the version
- read app credentials

## Error Handling

If the preferred callback port is unavailable and the random port also fails, return the bind error. If app creation or publishing fails, return the platform API error with the response snippet already provided by the platform client.

## Tests

Cover the behavior with unit tests:

- callback server falls back to a random local port when the preferred port is occupied
- platform setup threads the runtime callback URL into redirect configuration and OAuth URL generation
- platform client creates an app when list lookup does not find `lark_cli`
