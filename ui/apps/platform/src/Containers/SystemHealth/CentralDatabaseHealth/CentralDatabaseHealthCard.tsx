import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Content,
    Flex,
    FlexItem,
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
                <Flex className="pf-v6-u-flex-grow-1">
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
                        <Content component="p" key={message}>
                            {message}
                        </Content>
                    ))}
                </CardBody>
            )}
            {error && (
                <CardBody>
                    <Content component="p">
                        There was an error querying the database status:
                    </Content>
                    <Content component="p">{getAxiosErrorMessage(error)}</Content>
                </CardBody>
            )}
        </Card>
    );
}

export default CentralDatabaseHealthCard;
