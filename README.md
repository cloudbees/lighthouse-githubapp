## Lighthouse GitHub App

This is a multi-tenant version of [Lighthouse](https://github.com/jenkins-x/lighthouse) for use in the SaaS which implements a GitHub App.

### Environment variables

The following environment variables are required if you want to run this app locally:

| Name  |  Description |
| ------------- | ------------- |
| `LHA_APP_ID` | The GitHub App ID (shown on the Apps page) |
| `LHA_HMAC_TOKEN` | The HMAC token to verify webhooks |
| `LHA_PRIVATE_KEY_FILE` | The location of the private key file from the GitHub App |
| `BOT_NAME` | optional name of the current bot. e.g. `myapp[bot]` |


### Building

Run

    go build && ./lighthouse-githubapp
    
    