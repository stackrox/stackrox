import React, { ReactElement } from 'react';
import {
    Bullseye,
    Button,
    ButtonVariant,
    EmptyState,
    EmptyStateBody,
    Title,
} from '@patternfly/react-core';

import ButtonLink from 'Components/PatternFly/ButtonLink';

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
                    <ButtonLink variant={ButtonVariant.link} isInline to={url}>
                        {actionText}
                    </ButtonLink>
                )}
            </EmptyState>
        </Bullseye>
    );
};

export default NotFoundMessage;
