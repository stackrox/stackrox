const dataResolversByCategory = {
    Traffic: (datum) => datum.traffic,
    Deployment: (datum) => datum.deploymentName,
    Namespace: (datum) => datum.namespace,
    Protocols: (datum) => datum.portsAndProtocols.map((d) => String(d.protocol)),
    Ports: (datum) => datum.portsAndProtocols.map((d) => String(d.port)),
    Connection: (datum) => datum.connection,
};

function getNetworkFlowValueByCategory(datum, category) {
    return dataResolversByCategory[category]?.(datum);
}

export default getNetworkFlowValueByCategory;
