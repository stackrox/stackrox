import entityTypes from 'constants/entityTypes';
import resolvePath from 'object-resolve-path';
import isEmpty from 'lodash/isEmpty';

const entityNameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: data => resolvePath(data, 'serviceAccount.name'),
    [entityTypes.SECRET]: data => resolvePath(data, 'secret.name'),
    [entityTypes.CLUSTER]: data => resolvePath(data, 'cluster.name'),
    [entityTypes.CVE]: data => resolvePath(data, 'vulnerability.cve'),
    [entityTypes.DEPLOYMENT]: data => resolvePath(data, 'deployment.name'),
    [entityTypes.NAMESPACE]: data => resolvePath(data, 'namespace.metadata.name'),
    [entityTypes.ROLE]: data => {
        if (!data || !data.clusters || !data.clusters.length) return null;
        const result = data.clusters.reduce((acc, curr) => {
            if (!curr.k8srole) return acc;
            return curr.k8srole.name;
        }, null);
        return result;
    },
    [entityTypes.NODE]: data => resolvePath(data, 'node.name'),
    [entityTypes.CONTROL]: data => {
        if (!data.control) return null;
        return `${data.control.name} - ${data.control.description}`;
    },
    [entityTypes.IMAGE]: data => resolvePath(data, 'image.name.fullName'),
    [entityTypes.POLICY]: data => resolvePath(data, 'policy.name'),
    [entityTypes.SUBJECT]: data => {
        if (!data || !data.clusters || !data.clusters.length) return null;
        const result = data.clusters.reduce((acc, curr) => {
            if (!curr.subject) return acc;
            return curr.subject.subject.name;
        }, null);
        return result;
    }
};

const getEntityName = (entityType, data, id) => {
    if (isEmpty(data)) return null;
    try {
        return entityNameKeyMap[entityType](data, id);
    } catch (error) {
        throw new Error(
            `Entity (${entityType}) is not mapped correctly in the "entityToNameResolverMapping"`
        );
    }
};

export default getEntityName;
