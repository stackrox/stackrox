Tag for [build #{{.Build.Number}}]({{.Build.URL}}) is `{{.Env.TAG}}`.

💻 For deploying this image using the dev scripts, run the following first:

```sh
export MAIN_IMAGE_TAG='{{.Env.TAG}}'
```

📦 You can also generate an installation bundle with:

```sh
docker run -i --rm stackrox/main:{{.Env.TAG}} central generate interactive > bundle.zip
```

🕹️ A `roxctl` binary artifact can be [downloaded from CircleCI](https://circleci.com/gh/stackrox/rox/{{.Build.Number}}#artifacts/containers/0).
