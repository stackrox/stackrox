import groups.End
import services.ImageService

import org.junit.experimental.categories.Category

@Category(End)
class UsedImagesTest extends BaseSpecification {

    public static final Set<String> ALLOWED_IMAGES = [
            'docker.io/docker/kube-compose-controller:v0.4.23',
            'docker.io/istio/proxyv2' +
                    '@sha256:134e99aa9597fdc17305592d13add95e2032609d23b4c508bd5ebd32ed2df47d',
            'docker.io/library/busybox:latest',
            'docker.io/library/busybox:latest',
            'docker.io/library/mysql' +
                    '@sha256:de2913a0ec53d98ced6f6bd607f487b7ad8fe8d2a86e2128308ebf4be2f92667',
            'docker.io/library/mysql' +
                    '@sha256:f7985e36c668bb862a0e506f4ef9acdd1254cdf690469816f99633898895f7fa',
            'docker.io/library/nginx:1.10',
            'docker.io/library/nginx' +
                    '@sha256:63aa22a3a677b20b74f4c977a418576934026d8562c04f6a635f0e71e0686b6d',
            'docker.io/library/nginx:latest',
            'docker.io/library/ubuntu:latest' +
                    '@sha256:3235326357dfb65f1781dbc4df3b834546d8bf914e82cce58e6e6b676e23ce8f',
            'docker.io/library/ubuntu:latest' +
                    '@sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4',
            'gcr.io/gke-release/nvidia-gpu-device-plugin@' +
                    'sha256:d6cb575b0d8a436066a0d3a783bbaf84697e0d5a68857edfe5fd5d1183133c7d',
            'gcr.io/distroless/base' +
                    '@sha256:bc217643f9c04fc8131878d6440dd88cf4444385d45bb25995c8051c29687766',
            'gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init' +
                    '@sha256:79f768d28ff9af9fcbf186f9fc1b8e9f88835dfb07be91610a1f17cf862db89e',
            'gke.gcr.io/calico/node:v3.23.3-gke.1',
            'gke.gcr.io/calico/typha:v3.23.3-gke.1',
            'gke.gcr.io/cluster-proportional-autoscaler:1.8.4-gke.1',
            'gke.gcr.io/cluster-proportional-autoscaler:1.8.5-gke.0',
            'gke.gcr.io/cpvpa:v0.8.3-gke.1',
            'gke.gcr.io/csi-node-driver-registrar:v2.5.0-gke.1',
            'gke.gcr.io/gke-metrics-agent:1.8.3-gke.0',
            'gke.gcr.io/ingress-gce-404-server-with-metrics:v1.13.4',
            'gke.gcr.io/ip-masq-agent:v2.8.0',
            'gke.gcr.io/k8s-dns-dnsmasq-nanny:1.22.9-gke.0',
            'gke.gcr.io/k8s-dns-kube-dns:1.22.9-gke.0',
            'gke.gcr.io/k8s-dns-sidecar:1.22.9-gke.0',
            'gke.gcr.io/prometheus-to-sd:v0.11.3-gke.0',
            'gke.gcr.io/proxy-agent:v0.0.31-gke.0',
            'us.gcr.io/stackrox-ci/nginx:1.10.2',
            'us.gcr.io/stackrox-ci/nginx:1.11',
            'us.gcr.io/stackrox-ci/nginx:1.11.1',
            'us.gcr.io/stackrox-ci/nginx:1.9.1',
            'us.gcr.io/stackrox-ci/qa/fail-compliance/ssh:0.1',
            'us.gcr.io/stackrox-ci/qa/trigger-policy-violations/alpine:0.6',
            'us.gcr.io/stackrox-ci/qa/trigger-policy-violations/more:0.3',
            'us.gcr.io/stackrox-ci/qa/trigger-policy-violations/most:0.19',
            'gke.gcr.io/addon-resizer:1.8.14-gke.3',
            'gke.gcr.io/event-exporter:v0.3.10-gke.0',
            'gke.gcr.io/fluent-bit-gke-exporter:v0.22.0-gke.0',
            'gke.gcr.io/fluent-bit:v1.8.7-gke.5',
            'gke.gcr.io/metrics-server:v0.5.2-gke.1',
    ].toSet()

    def 'Verify images used for testing'() {
        when:
        Set<String> usedImages = ImageService.getImages()*.name.findAll { !it.startsWith('quay.io') }.toSet()

        log.info("used images ${usedImages}")

        then:
        def unusedImages = (ALLOWED_IMAGES - usedImages).sort()
        assert unusedImages.empty, "Please remove unused images: ${unusedImages}"
        and:
        def newImages = (usedImages - ALLOWED_IMAGES).sort()
        assert newImages.empty, "Consider using different image from allowed list: ${newImages}"
    }
}
