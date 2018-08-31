import { schema } from 'normalizr';

export const policy = new schema.Entity('policy');
export const deployment = new schema.Entity('deployment');

// Note: alert entitiy contains a reference to a policy, but it's a version of policy (potentially obsolete)
// at the time when alert fired. Therefore we don't specify policy ref in alert schema to not overwrite
// non-obsolete list of policies.
export const alert = new schema.Entity('alert');

export const alerts = { alerts: [alert] };

export const image = new schema.Entity('image', {}, { idAttribute: 'sha' });

export const cluster = new schema.Entity('cluster');

const getSecretId = secretObj => {
    if (secretObj.id) {
        return secretObj.id;
    }
    return secretObj.secret.id;
};

const flattenSecretObj = secretObj => {
    if (secretObj.secret) {
        return {
            id: secretObj.secret.id,
            name: secretObj.secret.name,
            cluster: secretObj.secret.clusterName,
            namespace: secretObj.secret.namespace,
            deploymentRelationships: secretObj.relationship.deploymentRelationships
        };
    }
    return {
        id: secretObj.id,
        name: secretObj.name,
        cluster: secretObj.clusterName,
        namespace: secretObj.namespace
    };
};

export const secret = new schema.Entity(
    'secret',
    {},
    {
        idAttribute: getSecretId,
        processStrategy: flattenSecretObj
    }
);
