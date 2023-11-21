import React, { ReactElement } from 'react';
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

import pluralize from 'pluralize';
import IntegrationsHealth from './IntegrationsHealth';
import { IntegrationMergedItem } from '../utils/integrations';
import { ErrorIcon, healthIconMap, SpinnerIcon } from '../CardHeaderIcons';

type IntegrationHealthWidgetProps = {
    integrationText: string;
    integrationsMerged: IntegrationMergedItem[];
    errorMessageFetching: string;
    isFetchingInitialRequest: boolean;
};

const IntegrationHealthWidget = ({
    integrationText,
    integrationsMerged,
    errorMessageFetching,
    isFetchingInitialRequest,
}: IntegrationHealthWidgetProps): ReactElement => {
    const integrations = integrationsMerged.filter((integrationMergedItem) => {
        return integrationMergedItem.status === 'UNHEALTHY';
    });
    /* eslint-disable no-nested-ternary */
    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : errorMessageFetching
          ? ErrorIcon
          : healthIconMap[integrations.length === 0 ? 'success' : 'danger'];
    /* eslint-enable no-nested-ternary */
    const hasCount = !isFetchingInitialRequest && !errorMessageFetching;

    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                <CardHeaderMain>
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
                    </Flex>
                </CardHeaderMain>
            </CardHeader>
            {(errorMessageFetching || integrations.length !== 0) && (
                <CardBody>
                    {errorMessageFetching ? (
                        <Alert isInline variant="warning" title={errorMessageFetching} />
                    ) : (
                        <IntegrationsHealth integrations={integrations} />
                    )}
                </CardBody>
            )}
        </Card>
    );
};

export default IntegrationHealthWidget;
