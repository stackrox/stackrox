import React from 'react';
import {
    Bullseye,
    EmptyState,
    EmptyStateVariant,
    EmptyStateIcon,
    Title,
    EmptyStateBody,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type ErrorSectionProps = {
    error: Error;
    children?: React.ReactNode;
};

function ErrorSection({ error, children }: ErrorSectionProps) {
    return (
        <Bullseye>
            <EmptyState variant={EmptyStateVariant.large}>
                <EmptyStateIcon className="pf-u-danger-color-100" icon={ExclamationCircleIcon} />
                <Title headingLevel="h2">{getAxiosErrorMessage(error)}</Title>
                {children && <EmptyStateBody>{children}</EmptyStateBody>}
            </EmptyState>
        </Bullseye>
    );
}

export default ErrorSection;
