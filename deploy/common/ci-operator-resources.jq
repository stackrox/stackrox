# Older operator charts do not understand the CI resource-policy annotation.
# Remove CPU limits after their chart defaulting using the existing overlay API.
def cpu_overlay($kind; $name; $containers; $init):
  {apiVersion: "apps/v1", kind: $kind, name: $name, optional: true,
   patches: ([$containers[] | {path: ("spec.template.spec.containers[name:" + . + "].resources.limits.cpu")}] +
             [$init[] | {path: ("spec.template.spec.initContainers[name:" + . + "].resources.limits.cpu")}])};
def cpu_overlays($kind):
  if $kind == "Central" then
    [cpu_overlay("Deployment"; "central"; ["central"]; []),
     cpu_overlay("Deployment"; "central-worker"; ["central-worker"]; []),
     cpu_overlay("Deployment"; "config-controller"; ["manager"]; []),
     cpu_overlay("Deployment"; "central-db"; ["central-db"]; ["init-db"]),
     cpu_overlay("Deployment"; "scanner-v4-matcher"; ["matcher"]; []),
     cpu_overlay("Deployment"; "scanner-v4-indexer"; ["indexer"]; []),
     cpu_overlay("Deployment"; "scanner-v4-db"; ["db"]; ["init-db"])]
  else
    [cpu_overlay("Deployment"; "sensor"; ["sensor"]; []),
     cpu_overlay("Deployment"; "admission-control"; ["admission-control"]; []),
     cpu_overlay("DaemonSet"; "collector"; ["collector", "compliance"]; []),
     cpu_overlay("Deployment"; "scanner-v4-indexer"; ["indexer"]; []),
     cpu_overlay("Deployment"; "scanner-v4-db"; ["db"]; ["init-db"])]
  end +
  [cpu_overlay("Deployment"; "scanner"; ["scanner"]; []),
   cpu_overlay("Deployment"; "scanner-db"; ["db"]; ["init-db"])];
def configure($kind):
  .customize.annotations["ci.stackrox.io/resource-policy"] = "requests" |
  .overlays as $existing |
  .overlays = (($existing // []) + (cpu_overlays($kind) - ($existing // [])));
if .kind == "Central" or .kind == "SecuredCluster" then
  .kind as $kind | .spec |= configure($kind)
elif has("central") or has("securedCluster") then
  .central.spec |= configure("Central") |
  .securedCluster.spec |= configure("SecuredCluster")
else . end
