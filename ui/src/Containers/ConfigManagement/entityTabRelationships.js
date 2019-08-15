import entityTypes from 'constants/entityTypes';

const entityTabsMap = {
    [entityTypes.SERVICE_ACCOUNT]: [entityTypes.DEPLOYMENT, entityTypes.ROLE],
    [entityTypes.ROLE]: [entityTypes.SUBJECT, entityTypes.SERVICE_ACCOUNT],
    [entityTypes.SECRET]: [entityTypes.DEPLOYMENT],
    [entityTypes.CLUSTER]: [
        entityTypes.NODE,
        entityTypes.SECRET,
        entityTypes.IMAGE,
        entityTypes.NAMESPACE,
        entityTypes.DEPLOYMENT,
        entityTypes.SUBJECT,
        entityTypes.SERVICE_ACCOUNT,
        entityTypes.ROLE,
        entityTypes.POLICY
    ],
    [entityTypes.NAMESPACE]: [
        entityTypes.DEPLOYMENT,
        entityTypes.SECRET,
        entityTypes.IMAGE,
        entityTypes.POLICY
    ],
    [entityTypes.NODE]: [entityTypes.CONTROL],
    [entityTypes.IMAGE]: [entityTypes.DEPLOYMENT],
    [entityTypes.CONTROL]: [entityTypes.NODE],
    [entityTypes.SUBJECT]: [entityTypes.ROLE],
    [entityTypes.DEPLOYMENT]: [entityTypes.IMAGE, entityTypes.SECRET, entityTypes.POLICY],
    [entityTypes.POLICY]: [entityTypes.DEPLOYMENT]
};

export default entityTabsMap;
