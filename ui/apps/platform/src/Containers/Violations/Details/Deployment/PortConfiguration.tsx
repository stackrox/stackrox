import { Fragment } from 'react';
import type { ReactElement } from 'react';
import { Card, CardBody, CardTitle, Title } from '@patternfly/react-core';

import type { Deployment } from 'types/deployment.proto';
import PortDescriptionList from './PortDescriptionList';

export type PortConfigurationProps = {
    deployment: Deployment | null;
};

function PortConfiguration({ deployment }: PortConfigurationProps): ReactElement {
    let content: JSX.Element[] | string = 'None';

    if (deployment === null) {
        content =
            'Port configurations are unavailable because the alert’s deployment no longer exists.';
    } else if (deployment.ports.length !== 0) {
        content = deployment.ports.map((port, i) => {
            /* eslint-disable react/no-array-index-key */
            return (
                <Fragment key={i}>
                    <Title headingLevel="h4" className="pf-v5-u-mb-md">{`ports[${i}]`}</Title>
                    <PortDescriptionList port={port} />
                </Fragment>
            );
            /* eslint-enable react/no-array-index-key */
        });
    }

    return (
        <Card isFlat>
            <CardTitle component="h3">Port configuration</CardTitle>
            <CardBody>{content}</CardBody>
        </Card>
    );
}

export default PortConfiguration;
