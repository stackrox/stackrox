import React from 'react';
import { Alert } from 'types/alert.proto';

export type MostRecentViolationsProps = {
    alerts: Partial<Alert>[];
};

function MostRecentViolations({ alerts }: MostRecentViolationsProps) {
    return <>{JSON.stringify(alerts)}</>;
}

export default MostRecentViolations;
