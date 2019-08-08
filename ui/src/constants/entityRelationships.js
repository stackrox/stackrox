import entityTypes from 'constants/entityTypes';

const parents = {
    [entityTypes.IMAGE]: entityTypes.DEPLOYMENT,
    [entityTypes.DEPLOYMENT]: entityTypes.NAMESPACE,
    [entityTypes.NAMESPACE]: entityTypes.CLUSTER,
    [entityTypes.SERVICE_ACCOUNT]: entityTypes.NAMESPACE,
    [entityTypes.SECRET]: entityTypes.NAMESPACE,
    [entityTypes.ROLE]: entityTypes.CLUSTER,
    [entityTypes.NODE]: entityTypes.CLUSTER
};

export default parents;
