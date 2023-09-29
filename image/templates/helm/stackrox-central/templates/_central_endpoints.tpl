{{ define "srox.configureCentralEndpoints" }}
{{ $central := . }}
{{ $containerPorts := list (dict "name" "api" "containerPort" 8443) }}
{{ $netPolIngressRules := list (dict "ports" (list (dict "port" 8443 "protocol" "TCP"))) }}
{{ $servicePorts := list (dict "name" "https" "targetPort" "api" "port" 443) }}
{{ $cfgDict := fromYaml $central._endpointsConfig }}
{{ if kindIs "map" $cfgDict }}
  {{ if $cfgDict.disableDefault }}
    {{ $containerPorts = list }}
    {{ $netPolIngressRules = list }}
    {{ $servicePorts = list }}
  {{ end }}
  {{ range $epCfg := default list $cfgDict.endpoints }}
    {{ if and $epCfg.listen (kindIs "string" $epCfg.listen) }}
      {{ $listenParts := splitList ":" $epCfg.listen }}
      {{ if $listenParts }}
        {{ $port := last $listenParts }}
        {{ if $port }}
          {{ if regexMatch "[0-9]+" $port }}
            {{ $port = int $port }}
          {{ end }}
          {{ $containerPort := dict "containerPort" $port }}
          {{ if and $epCfg.name (kindIs "string" $epCfg.name) }}
            {{ $_ := set $containerPort "name" $epCfg.name }}
          {{ end }}
          {{ $containerPorts = append $containerPorts $containerPort }}
          {{ if $epCfg.servicePort }}
            {{ $servicePort := dict "targetPort" $port "port" $epCfg.servicePort }}
            {{ if $containerPort.name }}
              {{ $_ := set $servicePort "name" $containerPort.name }}
            {{ end }}
            {{ $servicePorts = append $servicePorts $servicePort }}
          {{ end }}
          {{ if not (kindIs "invalid" $epCfg.allowIngressFrom) }}
            {{ $fromList := $epCfg.allowIngressFrom }}
            {{ if not (kindIs "slice" $fromList) }}
              {{ $fromList = list $fromList }}
            {{ end }}
            {{ $netPolIngressRule := dict "ports" (list (dict "port" $port "protocol" "TCP")) "from" $fromList }}
            {{ $netPolIngressRules = append $netPolIngressRules $netPolIngressRule }}
          {{ end }}
        {{ end }}
      {{ end }}
    {{ end }}
  {{ end }}
{{ end }}
{{ if $central.exposeMonitoring }}
  {{ $containerPorts = append $containerPorts (dict "name" "monitoring" "containerPort" 9090) }}
  {{ $servicePorts = append $servicePorts (dict "name" "monitoring" "targetPort" "monitoring" "port" 9090) }}
{{ end }}
# The (...) safe-guard against nil pointer evaluations for Helm versions built with Go < 1.18.
{{ if ((($central.monitoring).openshift).enabled) }}
  {{ $containerPorts = append $containerPorts (dict "name" "monitoring-tls" "containerPort" 9091) }}
  {{ $servicePorts = append $servicePorts (dict "name" "monitoring-tls" "targetPort" "monitoring-tls" "port" 9091) }}
{{ end }}
{{ $_ := set $central "_containerPorts" $containerPorts }}
{{ $_ = set $central "_servicePorts" $servicePorts }}
{{ $_ = set $central "_netPolIngressRules" $netPolIngressRules }}
{{ end }}
