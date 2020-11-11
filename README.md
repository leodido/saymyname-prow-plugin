# saymyname-prow-plugin

This is a [Prow](https://github.com/kubernetes/test-infra/tree/master/prow) [external plugin](https://github.com/kubernetes/test-infra/tree/master/prow/plugins#external-plugins).

> If you comment `/poiana` on Github, Prow replies with a random sentence...

![](screenshot.png)

You can learn about Prow external plugins from below links:

- [Going Beyond CI/CD with Prow at KubeCon EU 2020 (video)](https://youtu.be/qQvoImxHydk?t=1554)
- [Going Beyond CI/CD with Prow at KubeCon EU 2020 (slides)](https://speakerdeck.com/leodido/cd-with-prow)
- [custom external plugin](https://github.com/kubernetes/test-infra/tree/master/prow/plugins#external-plugins)
- [in-cluster plugins](https://github.com/kubernetes/test-infra/tree/master/prow/plugins)
- [official external plugins](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins)

Docker image is here: https://hub.docker.com/repository/docker/leodido/saymyname-prow-plugin

## Deploy plugin

```
$ kubectl apply -f https://raw.githubusercontent.com/leodido/prow-plugin-saymyname/master/deploy.yaml
```

## Enable plugin

Append a below setting to your `plugins.yaml`.

```
external_plugins:
  <org>/<repo>:
  - name: saymyname
    endpoint: http://saymyname.default.svc.cluster.local:8787
    events:
    - issue_comment
```

## TODO

- [ ] Make configurable:
  - [ ] Slash command
  - [ ] Sentences set
- [ ] Test it with [phony](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/phony#phony)
