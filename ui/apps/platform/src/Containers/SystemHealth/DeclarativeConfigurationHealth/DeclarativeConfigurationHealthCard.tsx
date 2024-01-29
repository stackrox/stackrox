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
import pluralize from 'pluralize';

import { fetchDeclarativeConfigurationsHealth } from 'services/DeclarativeConfigHealthService';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { ErrorIcon, healthIconMap, SpinnerIcon } from '../CardHeaderIcons';
import { DeclarativeConfigHealth } from '../../../types/declarativeConfigHealth.proto';

type DeclarativeConfigurationHealthCardProps = {
    pollingCount: number;
};

function DeclarativeConfigurationHealthCard({
    pollingCount,
}: DeclarativeConfigurationHealthCardProps): ReactElement {
    const [isFetching, setIsFetching] = useState(false);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [items, setItems] = useState<DeclarativeConfigHealth[]>([]);

    useEffect(() => {
        setIsFetching(true);
        fetchDeclarativeConfigurationsHealth()
            .then((itemsFetched) => {
                setErrorMessageFetching('');
                setItems(itemsFetched.response.healths);
            })
            .catch((error) => {
                setErrorMessageFetching(getAxiosErrorMessage(error));
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
    const isFetchingInitialRequest = isFetching && pollingCount === 0;
    const hasCount = !isFetchingInitialRequest && !errorMessageFetching;
    const unhealthyItems = items.filter(({ status }) => status === 'UNHEALTHY');
    const unhealthyCount = unhealthyItems.length;

    /* eslint-disable no-nested-ternary */
    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : errorMessageFetching
          ? ErrorIcon
          : healthIconMap[unhealthyCount === 0 ? 'success' : 'danger'];
    /* eslint-enable no-nested-ternary */

    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                <CardHeaderMain>
                    <Flex className="pf-u-flex-grow-1">
                        <FlexItem>{icon}</FlexItem>
                        <FlexItem>
                            <CardTitle component="h2">Declarative configuration</CardTitle>
                        </FlexItem>
                        {hasCount && (
                            <FlexItem>
                                {unhealthyCount === 0
                                    ? 'no errors'
                                    : `${unhealthyCount} ${pluralize('error', unhealthyCount)}`}
                            </FlexItem>
                        )}
                    </Flex>
                </CardHeaderMain>
            </CardHeader>
            {(errorMessageFetching || unhealthyCount !== 0) && (
                <CardBody>
                    {errorMessageFetching ? (
                        <Alert isInline variant="warning" title={errorMessageFetching} />
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
