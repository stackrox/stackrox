import React, { ReactElement } from 'react';
import {
    Bullseye,
    Button,
    ButtonVariant,
    EmptyState,
    EmptyStateBody, EmptyStateHeader, EmptyStateFooter,
    } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

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
                <EmptyStateHeader titleText={<>{title}</>} headingLevel="h1" /><EmptyStateFooter>
                {message && <EmptyStateBody>{message}</EmptyStateBody>}
                {isButtonVisible && <Button variant="primary">{actionText}</Button>}
                {isLinkVisible && (
                    <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
                        {actionText}
                    </Button>
                )}
            </EmptyStateFooter></EmptyState>
        </Bullseye>
    );
};

export default NotFoundMessage;
