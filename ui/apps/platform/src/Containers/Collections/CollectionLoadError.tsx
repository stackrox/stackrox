import React from 'react';
import { Bullseye, Title } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type CollectionLoadErrorProps = {
    title: string;
    error: Error;
};

function ErrorIcon(props: SVGIconProps) {
    return (
        <ExclamationCircleIcon
            {...props}
            style={{ color: 'var(--pf-global--danger-color--200)' }}
        />
    );
}

function CollectionLoadError({ title, error }: CollectionLoadErrorProps) {
    return (
        <Bullseye>
            <EmptyStateTemplate title={title} headingLevel="h2" icon={ErrorIcon}>
                <Title headingLevel="h3">{getAxiosErrorMessage(error)}</Title>
            </EmptyStateTemplate>
        </Bullseye>
    );
}

export default CollectionLoadError;
