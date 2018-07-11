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

const getSecretId = secretObj => secretObj.secret.id;

const flattenSecretObj = secretObj => ({
    id: secretObj.secret.id,
    name: secretObj.secret.name,
    clusterRelationship: secretObj.relationship.clusterRelationship,
    namespaceRelationship: secretObj.relationship.namespaceRelationship,
    deploymentRelationships: secretObj.relationship.deploymentRelationships
});

export const secret = new schema.Entity(
    'secret',
    {},
    {
        idAttribute: getSecretId,
        processStrategy: flattenSecretObj
    }
);
