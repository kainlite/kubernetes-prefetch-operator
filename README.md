### Forward Operator

[![pipeline status](https://gitlab.com/kainlite/kubernetes-prefetch-operator/badges/master/pipeline.svg)](https://gitlab.com/kainlite/kubernetes-prefetch-operator/commits/master)
[![coverage report](https://gitlab.com/kainlite/kubernetes-prefetch-operator/badges/master/coverage.svg)](https://gitlab.com/kainlite/kubernetes-prefetch-operator/commits/master)

This project aims to do one thing: Prefetch your docker images in all your nodes given a deployment and some filters.

There is a blog page describing how to get here, [check it out](https://techsquad.rocks/blog/creating_a_prefetch_method_for_kubernetes_with_the_operator_sdk/).

### Installation
To install this operator in your cluster you need to do the following:
```
make deploy IMG=kubernetes/kainlite:latest
```

### Use cases
If you have large images (anti-pattern) or just want to speed things up a bit at runtime then this might help, also it's an interesting learning weekend project.

<a href="https://www.buymeacoffee.com/NDx5OFh" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-green.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" ></a>
