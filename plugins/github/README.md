# GitHub plugin

This plugin provides interactions with GitHub.

Currently only one Service `github.webhook` is provided. Configuration is as follows:

```yaml
path: /path/to/local/repository
branch: gh-pages
secret: # if provided, will try to verify the HMAC signature of the webhook
```

It listens for incoming GitHub webhooks in JSON, and runs `git fetch <branch> && git reset --hard FETCH_HEAD` in the specified path.
