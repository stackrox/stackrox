import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import {
    Alert,
    Card,
    CardFooter,
    CardHeader,
    CardHeaderMain,
    CardActions,
    CardTitle,
    CardBody,
} from '@patternfly/react-core';

import { integrationsPath } from 'routePaths';
import ViewAllButton from 'Components/PatternFly/ViewAllButton';
import { styleHealthyPF, styleUnhealthyPF } from 'Containers/Clusters/cluster.helpers';
import IntegrationsHealth from './IntegrationsHealth';
import { splitIntegrationsByHealth, IntegrationMergedItem } from '../utils/integrations';

type IntegrationHealthWidgetProps = {
    id: string;
    integrationText: string;
    integrationsMerged: IntegrationMergedItem[];
    requestHasError: boolean;
};

const IntegrationHealthWidget = ({
    id,
    integrationText,
    integrationsMerged,
    requestHasError,
}: IntegrationHealthWidgetProps): ReactElement => {
    const numIntegrations = integrationsMerged.length;

    const { HEALTHY: healthy, UNHEALTHY: unhealthy } =
        splitIntegrationsByHealth(integrationsMerged);
    const numHealthly = healthy.length;

    let text = 'No configured integrations';
    if (numIntegrations !== 0) {
        if (numHealthly === numIntegrations) {
            text = `${numHealthly} healthy ${pluralize('integration', numHealthly)}`;
        } else {
            text = `${numHealthly} / ${numIntegrations} healthy integrations`; // cannot both be singular
        }
    }

    const { Icon, fgColor } = unhealthy.length > 0 ? styleUnhealthyPF : styleHealthyPF;

    return (
        <Card isFullHeight isCompact>
            <CardHeader>
                <CardHeaderMain>
                    <CardTitle component="h2">{integrationText}</CardTitle>
                </CardHeaderMain>
                <CardActions>
                    <ViewAllButton url={`${integrationsPath}#${id}`} />
                </CardActions>
            </CardHeader>
            {requestHasError ? (
                <CardBody>
                    <Alert
                        isInline
                        variant="danger"
                        title={`Request failed for ${integrationText}`}
                    />
                </CardBody>
            ) : (
                <>
                    <CardBody>
                        <IntegrationsHealth
                            unhealthyIntegrations={unhealthy}
                            Icon={Icon}
                            fgColor={fgColor}
                        />
                    </CardBody>
                    <CardFooter>
                        <div
                            className={`pf-u-text-align-center ${fgColor}`}
                            data-testid="healthy-text"
                        >
                            {text}
                        </div>
                    </CardFooter>
                </>
            )}
        </Card>
    );
};

export default IntegrationHealthWidget;
