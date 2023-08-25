export type IntegrationHealthStatus = 'UNINITIALIZED' | 'UNHEALTHY' | 'HEALTHY';

// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// prettier-ignore
export type IntegrationHealthType =
    | 'UNKNOWN'
    | 'IMAGE_INTEGRATION'
    | 'NOTIFIER'
    | 'BACKUP'
    | 'DECLARATIVE_CONFIG'
    ;

export type IntegrationHealth = {
    id: string;
    name: string;
    type: IntegrationHealthType;
    status: IntegrationHealthStatus;
    errorMessage: string;
    lastTimestamp: string; // ISO 8601 timestamp when the status was ascertained
};
