export type DeclarativeConfigHealthStatus = 'UNHEALTHY' | 'HEALTHY';

export type DeclarativeConfigResourceType =
    | 'CONFIG_MAP'
    | 'ACCESS_SCOPE'
    | 'PERMISSION_SET'
    | 'ROLE'
    | 'AUTH_PROVIDER'
    | 'GROUP'
    | 'NOTIFIER';

export type DeclarativeConfigHealth = {
    id: string;
    name: string;
    status: DeclarativeConfigHealthStatus;
    resourceType: DeclarativeConfigResourceType;
    resourceName: string;
    errorMessage: string;
    lastTimestamp: string; // ISO 8601 timestamp when the status was ascertained.
};
