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
        <div className="pf-u-w-100 pf-u-h-100">
            <Bullseye>
                <EmptyState>
                    <Title headingLevel="h4" size="lg">
                        {title}
                    </Title>
                    {message && <EmptyStateBody>{message}</EmptyStateBody>}
                    {isButtonVisible && <Button variant="primary">{actionText}</Button>}
                    {isLinkVisible && (
                        <Link to={url}>
                            <Button className="pf-u-mt-lg" variant="primary">
                                {actionText}
                            </Button>
                        </Link>
                    )}
                </EmptyState>
            </Bullseye>
        </div>
    );
};

export default NotFoundMessage;
