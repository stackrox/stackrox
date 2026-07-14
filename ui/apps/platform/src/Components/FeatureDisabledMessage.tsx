import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Bullseye, EmptyState, EmptyStateBody, EmptyStateFooter } from '@patternfly/react-core';
import { BanIcon } from '@patternfly/react-icons';

export type FeatureDisabledMessageProps = {
    title: string;
    message?: string;
    actionText?: string;
    url?: string;
};

const FeatureDisabledMessage = ({
    title,
    message,
    actionText,
    url,
}: FeatureDisabledMessageProps): ReactElement => {
    const isLinkVisible = actionText && url;
    return (
        <Bullseye className="pf-v6-u-flex-grow-1">
            <EmptyState headingLevel="h1" titleText={title} icon={BanIcon} status="custom">
                <EmptyStateFooter>
                    {message && <EmptyStateBody>{message}</EmptyStateBody>}
                    {isLinkVisible && <Link to={url}>{actionText}</Link>}
                </EmptyStateFooter>
            </EmptyState>
        </Bullseye>
    );
};

export default FeatureDisabledMessage;
