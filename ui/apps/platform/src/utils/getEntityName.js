import entityTypes from 'constants/entityTypes';
import resolvePath from 'object-resolve-path';
import isEmpty from 'lodash/isEmpty';

const entityNameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: (data) => resolvePath(data, 'serviceAccount.name'),
    [entityTypes.SECRET]: (data) => resolvePath(data, 'secret.name'),
    [entityTypes.CLUSTER]: (data) => resolvePath(data, 'cluster.name'),
    [entityTypes.COMPONENT]: (data) => {
        const name = resolvePath(data, 'component.name');
        const version = resolvePath(data, 'component.version');

        return version ? `${name} ${version}` : name;
    },
    [entityTypes.CVE]: (data) => resolvePath(data, 'vulnerability.cve'),
    [entityTypes.IMAGE_CVE]: (data) => resolvePath(data, 'vulnerability.cve'),
    [entityTypes.NODE_CVE]: (data) => resolvePath(data, 'vulnerability.cve'),
    [entityTypes.CLUSTER_CVE]: (data) => resolvePath(data, 'vulnerability.cve'),
    [entityTypes.DEPLOYMENT]: (data) => resolvePath(data, 'deployment.name'),
    [entityTypes.NAMESPACE]: (data) => resolvePath(data, 'namespace.metadata.name'),
    [entityTypes.ROLE]: (data) => {
        if (!data || !data.clusters || !data.clusters.length) {
            return null;
        }
        const result = data.clusters.reduce((acc, curr) => {
            if (!curr.k8sRole) {
                return acc;
            }
            return curr.k8sRole.name;
        }, null);
        return result;
    },
    [entityTypes.NODE]: (data) => resolvePath(data, 'node.name'),
    [entityTypes.CONTROL]: (data) => {
        if (!data.control) {
            return null;
        }
        return `${data.control.name} - ${data.control.description}`;
    },
    [entityTypes.IMAGE]: (data) => resolvePath(data, 'image.name.fullName'),
    [entityTypes.POLICY]: (data) => resolvePath(data, 'policy.name'),
    [entityTypes.SUBJECT]: (data) => resolvePath(data, 'subject.name'),
};

function extractEntityName(entityType, data) {
    if (!data || isEmpty(data)) {
        return null;
    }

    const fn = entityNameKeyMap[entityType];
    if (fn) {
        return fn(data);
    }

    // No name extraction method defined. Make an educated guess.
    const firstKey = Object.keys(data)[0];
    return data[firstKey]?.name;
}

const getEntityName = (entityType, data) => {
    if (isEmpty(data)) {
        return null;
    }
    try {
        return extractEntityName(entityType, data);
    } catch (error) {
        throw new Error(
            `Entity (${entityType}) is not mapped correctly in the "entityToNameResolverMapping"`
        );
    }
};

export default getEntityName;
