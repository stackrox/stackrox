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
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import pluralize from 'pluralize';
import IntegrationsHealth from './IntegrationsHealth';
import { IntegrationMergedItem } from '../utils/integrations';
import IconText from '../../../Components/PatternFly/IconText/IconText';

type IntegrationHealthWidgetProps = {
    integrationText: string;
    integrationsMerged: IntegrationMergedItem[];
    errorMessageFetching: string;
};

const IntegrationHealthWidget = ({
    integrationText,
    integrationsMerged,
    errorMessageFetching,
}: IntegrationHealthWidgetProps): ReactElement => {
    const integrations = integrationsMerged.filter((integrationMergedItem) => {
        return integrationMergedItem.status === 'UNHEALTHY';
    });
    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                <CardHeaderMain>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem>
                            <CardTitle component="h2">{integrationText}</CardTitle>
                        </FlexItem>
                        {!errorMessageFetching && (
                            <FlexItem>
                                <IconText
                                    icon={
                                        integrations.length === 0 ? (
                                            <CheckCircleIcon color="var(--pf-global--success-color--100)" />
                                        ) : (
                                            <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />
                                        )
                                    }
                                    text={
                                        integrations.length === 0
                                            ? 'no errors'
                                            : `${integrations.length} ${pluralize(
                                                  'error',
                                                  integrations.length
                                              )}`
                                    }
                                />
                            </FlexItem>
                        )}
                    </Flex>
                </CardHeaderMain>
            </CardHeader>
            <CardBody>
                {errorMessageFetching ? (
                    <Alert isInline variant="warning" title={errorMessageFetching} />
                ) : (
                    <IntegrationsHealth integrations={integrations} />
                )}
            </CardBody>
        </Card>
    );
};

export default IntegrationHealthWidget;
