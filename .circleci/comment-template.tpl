Tag for [build #{{.Build.Number}}]({{.Build.URL}}) is `{{.Env.TAG}}`.

💻 For deploying this image using the dev scripts, run the following first:

```sh
export MAIN_IMAGE_TAG='{{.Env.TAG}}'
```

🕹️ A `roxctl` binary can be [downloaded from the CircleCI]({{.Build.URL}}) artifacts.
