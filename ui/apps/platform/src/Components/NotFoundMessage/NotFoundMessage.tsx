import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateFooter,
} from '@patternfly/react-core';

export type NotFoundMessageProps = {
    title: string;
    message?: string;
    actionText?: string;
    onClick?: () => void;
    url?: string;
};

const NotFoundMessage = ({
    title,
    message,
    actionText,
    onClick,
    url,
}: NotFoundMessageProps): ReactElement => {
    const isButtonVisible = actionText && onClick;
    const isLinkVisible = actionText && url;
    return (
        <Bullseye className="pf-v6-u-flex-grow-1">
            <EmptyState headingLevel="h1" titleText={title}>
                <EmptyStateFooter>
                    {message && <EmptyStateBody>{message}</EmptyStateBody>}
                    {isButtonVisible && (
                        <Button variant="primary" onClick={onClick}>
                            {actionText}
                        </Button>
                    )}
                    {isLinkVisible && <Link to={url}>{actionText}</Link>}
                </EmptyStateFooter>
            </EmptyState>
        </Bullseye>
    );
};

export default NotFoundMessage;
