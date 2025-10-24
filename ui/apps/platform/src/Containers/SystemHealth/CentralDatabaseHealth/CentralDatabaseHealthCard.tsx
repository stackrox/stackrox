import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Text,
} from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchDatabaseStatus } from 'services/DatabaseService';
import type { DatabaseStatus } from 'types/databaseService.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { ErrorIcon, SpinnerIcon, SuccessIcon, WarningIcon } from '../CardHeaderIcons';

function isDatabaseVersionSupported(databaseVersion: string | undefined): boolean {
    if (!databaseVersion) {
        return true;
    }
    return databaseVersion.startsWith('15');
}

function getDatabaseHealthInfo(data: DatabaseStatus | undefined): {
    status: 'healthy' | 'unhealthy' | undefined;
    messages: string[];
} {
    if (!data) {
        return { status: undefined, messages: [] };
    }

    // If the database is externally managed (ACSCS) then we do not currently have error information to report
    if (data.databaseIsExternal) {
        return { status: 'healthy', messages: [] };
    }

    if (!isDatabaseVersionSupported(data.databaseVersion)) {
        return {
            status: 'unhealthy',
            messages: [
                'Running an unsupported configuration of PostgreSQL',
                `Current version:  ${data.databaseVersion}`,
                `Required version: 15`,
            ],
        };
    }

    return { status: 'healthy', messages: [] };
}

function CentralDatabaseHealthCard(): ReactElement {
    const { data, isLoading, error } = useRestQuery(fetchDatabaseStatus);
    const databaseHealthInfo = getDatabaseHealthInfo(data);

    let icon = SpinnerIcon;

    if (isLoading) {
        icon = SpinnerIcon;
    } else if (error) {
        icon = ErrorIcon;
    } else if (databaseHealthInfo.status === 'unhealthy') {
        icon = WarningIcon;
    } else if (databaseHealthInfo.status === 'healthy') {
        icon = SuccessIcon;
    }

    return (
        <Card isCompact>
            <CardHeader>
                <Flex className="pf-v5-u-flex-grow-1">
                    <FlexItem>{icon}</FlexItem>
                    <FlexItem>
                        <CardTitle component="h2">Central database health</CardTitle>
                    </FlexItem>
                    {databaseHealthInfo.status && (
                        <FlexItem>
                            {databaseHealthInfo.status === 'healthy' ? 'no errors' : `warning`}
                        </FlexItem>
                    )}
                    {data?.databaseType && data.databaseVersion && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            {data.databaseType} {data.databaseVersion}
                        </FlexItem>
                    )}
                </Flex>
            </CardHeader>
            {databaseHealthInfo.status === 'unhealthy' && (
                <CardBody>
                    {databaseHealthInfo.messages.map((message) => (
                        <Text key={message}>{message}</Text>
                    ))}
                </CardBody>
            )}
            {error && (
                <CardBody>
                    <Text>There was an error querying the database status:</Text>
                    <Text>{getAxiosErrorMessage(error)}</Text>
                </CardBody>
            )}
        </Card>
    );
}

export default CentralDatabaseHealthCard;
