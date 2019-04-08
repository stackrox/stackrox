import uniq from 'lodash/uniq';
import entityTypes from 'constants/entityTypes';

export function getIndicesFromAggregatedResults(results) {
    if (!results || results.length === 0) return {};

    return results[0].aggregationKeys.reduce(
        (map, item, i) => ({ ...map, [item.scope]: i }),
        results[0].aggregationKeys[0],
        {}
    );
}

export function getResourceCountFromResults(type, data) {
    const { nodeResults, deploymentResults, namespaceResults, clusterResults } = data;
    let source;

    switch (type) {
        case entityTypes.NODE:
            source = nodeResults && nodeResults.results;
            break;
        case entityTypes.DEPLOYMENT:
            source = deploymentResults && deploymentResults.results;
            break;
        case entityTypes.NAMESPACE:
            source = namespaceResults && namespaceResults.results;
            break;
        case entityTypes.CLUSTER:
            source = clusterResults && clusterResults.results;
            break;
        default:
            source = clusterResults && clusterResults.results;
    }
    if (!source || source.length === 0) return 0;

    const index = getIndicesFromAggregatedResults(source, type)[type];
    if (!index && index !== 0) return 0;

    return uniq(
        source
            .filter(datum => datum.numFailing + datum.numPassing)
            .map(datum => datum.aggregationKeys[index].id)
    ).length;
}
