import entityTypes from 'constants/entityTypes';
import resolvePath from 'object-resolve-path';

const entityNameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: 'serviceAccount.name',
    [entityTypes.SECRET]: 'secret.name',
    [entityTypes.CLUSTER]: 'results.name',
    [entityTypes.DEPLOYMENT]: 'deployment.name',
    [entityTypes.NAMESPACE]: 'results.metadata.name',
    [entityTypes.ROLE]: 'clusters[0].k8srole.name'
};

const getEntityName = (entityType, data) => {
    const key = entityNameKeyMap[entityType];
    return resolvePath(data, key);
};

export default getEntityName;
