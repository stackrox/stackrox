import React, { ReactElement } from 'react';
import { Title, Button, EmptyState, EmptyStateBody, Bullseye } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

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
        <Bullseye className="pf-u-flex-grow-1">
            <EmptyState>
                <Title headingLevel="h4" size="lg">
                    {title}
                </Title>
                {message && <EmptyStateBody>{message}</EmptyStateBody>}
                {isButtonVisible && <Button variant="primary">{actionText}</Button>}
                {isLinkVisible && (
                    <Button
                        variant="link"
                        isInline
                        component={(props) => <Link {...props} to={url} />}
                    >
                        {actionText}
                    </Button>
                )}
            </EmptyState>
        </Bullseye>
    );
};

export default NotFoundMessage;
