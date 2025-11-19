import type { ConfigurationManagementEntityType } from 'utils/entityRelationships';

const entityTabsMap: Record<
    ConfigurationManagementEntityType,
    ConfigurationManagementEntityType[]
> = {
    SERVICE_ACCOUNT: ['DEPLOYMENT', 'ROLE'],
    ROLE: ['SUBJECT', 'SERVICE_ACCOUNT'],
    SECRET: ['DEPLOYMENT'],
    CLUSTER: [
        'CONTROL',
        'NODE',
        'SECRET',
        'IMAGE',
        'NAMESPACE',
        'DEPLOYMENT',
        'SUBJECT',
        'SERVICE_ACCOUNT',
        'ROLE',
    ],
    NAMESPACE: ['DEPLOYMENT', 'SECRET', 'IMAGE', 'SERVICE_ACCOUNT'],
    NODE: ['CONTROL'],
    IMAGE: ['DEPLOYMENT'],
    CONTROL: ['NODE'],
    SUBJECT: ['ROLE'],
    DEPLOYMENT: ['IMAGE', 'SECRET'],
    POLICY: ['DEPLOYMENT'],
};

export default entityTabsMap;
