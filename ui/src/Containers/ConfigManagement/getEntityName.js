import entityTypes from 'constants/entityTypes';
import resolvePath from 'object-resolve-path';
import isEmpty from 'lodash/isEmpty';

const entityNameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: data => resolvePath(data, 'serviceAccount.name'),
    [entityTypes.SECRET]: data => resolvePath(data, 'secret.name'),
    [entityTypes.CLUSTER]: data => resolvePath(data, 'results.name'),
    [entityTypes.DEPLOYMENT]: data => resolvePath(data, 'deployment.name'),
    [entityTypes.NAMESPACE]: data => resolvePath(data, 'results.metadata.name'),
    [entityTypes.ROLE]: data => resolvePath(data, 'clusters[0].k8srole.name'),
    [entityTypes.NODE]: data => resolvePath(data, 'node.name'),
    [entityTypes.CONTROL]: data => {
        if (!data.results) return null;
        return `${data.results.name} - ${data.results.description}`;
    },
    [entityTypes.IMAGE]: data => resolvePath(data, 'image.name.fullName')
};

const getEntityName = (entityType, data) => {
    if (isEmpty(data)) return null;
    return entityNameKeyMap[entityType](data);
};

export default getEntityName;
