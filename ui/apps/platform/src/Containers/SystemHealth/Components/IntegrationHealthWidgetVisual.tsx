import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Pagination,
} from '@patternfly/react-core';

import pluralize from 'pluralize';
import IntegrationsHealth from './IntegrationsHealth';
import type { IntegrationMergedItem } from '../utils/integrations';
import { ErrorIcon, SpinnerIcon, healthIconMap } from '../CardHeaderIcons';

type IntegrationHealthWidgetVisualProps = {
    integrationText: string;
    integrationsMerged: IntegrationMergedItem[];
    errorMessageFetching: string;
    isFetchingInitialRequest: boolean;
};

const IntegrationHealthWidgetVisual = ({
    integrationText,
    integrationsMerged,
    errorMessageFetching,
    isFetchingInitialRequest,
}: IntegrationHealthWidgetVisualProps): ReactElement => {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(10);

    function onSetPage(_, newPage) {
        setPage(newPage);
    }

    function onPerPageSelect(_, newPerPage) {
        setPerPage(newPerPage);
    }

    const integrations = integrationsMerged.filter((integrationMergedItem) => {
        return integrationMergedItem.status === 'UNHEALTHY';
    });

    const startIndex = (page - 1) * perPage;
    const paginatedIntegrations = integrations.slice(startIndex, startIndex + perPage);

    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : errorMessageFetching
          ? ErrorIcon
          : healthIconMap[integrations.length === 0 ? 'success' : 'danger'];

    const hasCount = !isFetchingInitialRequest && !errorMessageFetching;

    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                {
                    <>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <FlexItem>{icon}</FlexItem>
                            <FlexItem>
                                <CardTitle component="h2">{integrationText}</CardTitle>
                            </FlexItem>
                            {hasCount && (
                                <FlexItem>
                                    {integrations.length === 0
                                        ? 'no errors'
                                        : `${integrations.length} ${pluralize(
                                              'error',
                                              integrations.length
                                          )}`}
                                </FlexItem>
                            )}
                            {integrations.length > 0 && (
                                <FlexItem align={{ default: 'alignRight' }}>
                                    <Pagination
                                        itemCount={integrations.length}
                                        perPage={perPage}
                                        page={page}
                                        onSetPage={onSetPage}
                                        onPerPageSelect={onPerPageSelect}
                                    />
                                </FlexItem>
                            )}
                        </Flex>
                    </>
                }
            </CardHeader>
            {(errorMessageFetching || integrations.length !== 0) && (
                <CardBody>
                    {errorMessageFetching ? (
                        <Alert
                            isInline
                            variant="warning"
                            title={errorMessageFetching}
                            component="p"
                        />
                    ) : (
                        <IntegrationsHealth integrations={paginatedIntegrations} />
                    )}
                </CardBody>
            )}
        </Card>
    );
};

export default IntegrationHealthWidgetVisual;
