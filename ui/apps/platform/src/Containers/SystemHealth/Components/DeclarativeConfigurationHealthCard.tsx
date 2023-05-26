import React, { useState, useEffect, ReactElement } from 'react';
import {
    Alert,
    Card,
    CardBody,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import IconText from 'Components/PatternFly/IconText/IconText';
import { fetchDeclarativeConfigurationsHealth } from 'services/IntegrationHealthService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { IntegrationHealthItem } from '../utils/integrations';
import { getDateTime } from '../../../utils/dateUtils';

type CardProps = {
    pollingCount: number;
};

function DeclarativeConfigurationHealthCard({ pollingCount }: CardProps): ReactElement {
    const [isFetching, setIsFetching] = useState(false);
    const [requestErrorMessage, setRequestErrorMessage] = useState('');
    const [items, setItems] = useState<IntegrationHealthItem[]>([]);

    useEffect(() => {
        setIsFetching(true);
        fetchDeclarativeConfigurationsHealth()
            .then((itemsFetched) => {
                setRequestErrorMessage('');
                setItems(itemsFetched);
            })
            .catch((error) => {
                setRequestErrorMessage(getAxiosErrorMessage(error));
                setItems([]);
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);

    /*
     * Wait for isFetching only until response to the initial request.
     * Otherwise count temporarily disappears during each subsequent request.
     */
    const hasCount = (pollingCount !== 0 || !isFetching) && !requestErrorMessage;
    const unhealthyItems = items.filter((value) => {
        return value.status === 'UNHEALTHY';
    });
    const itemCount = unhealthyItems.length;

    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                <CardHeaderMain>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem>
                            <CardTitle component="h2">Declarative configuration</CardTitle>
                        </FlexItem>
                        {hasCount && (
                            <FlexItem>
                                <IconText
                                    icon={
                                        itemCount === 0 ? (
                                            <CheckCircleIcon color="var(--pf-global--success-color--100)" />
                                        ) : (
                                            <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />
                                        )
                                    }
                                    text={
                                        itemCount === 0
                                            ? 'no errors'
                                            : `${itemCount} ${pluralize('error', itemCount)}`
                                    }
                                />
                            </FlexItem>
                        )}
                    </Flex>
                </CardHeaderMain>
            </CardHeader>
            {(requestErrorMessage || itemCount !== 0) && (
                <CardBody>
                    {requestErrorMessage ? (
                        <Alert isInline variant="warning" title={requestErrorMessage} />
                    ) : (
                        <TableComposable variant="compact">
                            <Thead>
                                <Tr>
                                    <Th width={40}>Name</Th>
                                    <Th width={40}>Error</Th>
                                    <Th width={20}>Date</Th>
                                </Tr>
                            </Thead>
                            <Tbody data-testid="declarative-configs">
                                {unhealthyItems.map(({ id, name, errorMessage, lastTimestamp }) => (
                                    <Tr key={id}>
                                        <Td
                                            dataLabel="Name"
                                            modifier="breakWord"
                                            data-testid="integration-name"
                                        >
                                            {name}
                                        </Td>
                                        <Td
                                            dataLabel="Error"
                                            modifier="breakWord"
                                            data-testid="error-message"
                                        >
                                            {errorMessage}
                                        </Td>
                                        <Td dataLabel="Date">{getDateTime(lastTimestamp)}</Td>
                                    </Tr>
                                ))}
                            </Tbody>
                        </TableComposable>
                    )}
                </CardBody>
            )}
        </Card>
    );
}

export default DeclarativeConfigurationHealthCard;
