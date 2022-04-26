import { schema } from 'normalizr';

export const policy = new schema.Entity('policy');
export const deployment = new schema.Entity('deployment', undefined, {
    idAttribute: (value) => value.deployment.id,
});

export const image = new schema.Entity('image');

export const cluster = new schema.Entity('cluster');

export const secret = new schema.Entity('secret');
