import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import {
    Bullseye,
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateHeader,
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
        <Bullseye className="pf-v5-u-flex-grow-1">
            <EmptyState>
                <EmptyStateHeader titleText={title} headingLevel="h1" />
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
