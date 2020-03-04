## Lighthouse GitHub App

This is a multi-tenant version of [Lighthouse](https://github.com/jenkins-x/lighthouse) for use in the SaaS which implements a GitHub App.

### How it works

Here is a [diagram](https://whimsical.com/48NiENaA7vYCu8bUtgUfh8) of how it works:

![Overview](docs/images/app.png)


You can think of the Lighthouse GitHub App as like the regular Lighthouse - it handles webhooks from github, labels/comments on PRs and triggers pipelines - only it runs in a shared tenant rather than in each consumers cluster.

When the github app is installed to a github user/organisation all github webhooks for all repositories are sent to this HTTP endpoint.

Internally this service then queries the [jx-tenant-service](https://github.com/cloudbees/jx-tenant-service)'s REST API to query the workspaces and Scheduler JSON for the webhooks git URL.

Then for each webhook we:

* query the Workspace + Scheduler rows for the git URL
* for each Workspace + Scheduler:
  * connect to the remote Workspace project (for `KubeClient` / `JXClient` / `TektonClient` etc)
  * turn the `Scheduler` JSON into a lighthouse Prow `configs` and `plugins` configuration object
  * invoke the lighthouse webhook function [ProcessWebhook()](https://github.com/jenkins-x/lighthouse/blob/master/pkg/webhook/webhook.go#L233) to either comment on the PR or create a new pipeline in the tenant cluster via the metapipeline client.


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

    make build && ./build/lighthouse-githubapp
    
    
### Debugging in Dev/Staging

Ideally we would add support for [Stackdriver Debugging](https://cloud.google.com/debugger/) so we can easily debug stuff in production - however this is currently blocked on kaniko having issues building go source & crashing nodes. Until we figure that out, `telepresence` is a handy tool for debugging as its not always super easy to test out lighthouse-githubapp on a real cluster using real apps.    

* install [telepresence](https://www.telepresence.io/reference/install)
* connect to the Dev / Staging cluster where `lighthouse-githubapp` usually runs
* copy the google service account JSON to the file `https://www.telepresence.io/reference/install` which is usually inside the secret `jenkins-x-lighthouse-githubapp-saas` 
* start the debugger:

```
export BOT_NAME="jenkins-x[bot]"    
telepresence --swap-deployment jenkins-x-lighthouse-githubapp --expose 8080 --run dlv --listen=:2345 --headless=true --api-version=2 exec `which lighthouse-githubapp`
```

* now run the debug in your IDE using the usual remote debug Go option in your IDE. In IDEA/Goland you need to setup a `Go Remote` using the same port above `2345`


When you terminate the process/debug session `telepresence` will now switch back to the regular deployment again. You can force this to happen via:

```
sudo killall lighthouse-githubapp
sudo killall dlv
```   
