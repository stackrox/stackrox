import React, { ErrorInfo, ReactElement } from 'react';
import {
    EmptyState,
    EmptyStateIcon,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import ErrorBoundaryCodeBlock from './ErrorBoundaryCodeBlock';

import './ErrorBoundaryPage.css';

export type ErrorBoundaryPageProps = {
    error: Error;
    errorInfo: ErrorInfo;
};

/*
 * After an application page has thrown an error, render this page instead.
 *
 * The following remain the same from the application page:
 * Browser page address and title
 * Sidebar navigation active link
 *
 * spaceItemsXl is same as padding of EmptyState element.
 * spaceItemsSm separates heading from code block.
 *
 * flex_1 so each error-boundary-stack has equal width.
 */
function ErrorBoundaryPage({ error, errorInfo }: ErrorBoundaryPageProps): ReactElement {
    return (
        <PageSection id="error-boundary-page" variant="light">
            <Flex
                className="error-boundary-page-column"
                direction={{ default: 'column' }}
                flexWrap={{ default: 'nowrap' }}
                spaceItems={{ default: 'spaceItemsXl' }}
            >
                <EmptyState>
                    <EmptyStateIcon
                        icon={ExclamationCircleIcon}
                        color="var(--pf-global--danger-color--200)"
                    />
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Title headingLevel="h1">Cannot display the page</Title>
                        <p>The error has been logged.</p>
                    </Flex>
                </EmptyState>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Title headingLevel="h2">Error message</Title>
                    <ErrorBoundaryCodeBlock
                        code={error.message}
                        idForButton="error-boundary-button-error-message"
                        idForContent="error-boundary-content-error-message"
                        phraseForCopied="Copied to clipboard: Error message"
                        phraseForCopy="Copy to clipboard: Error message"
                    />
                </Flex>
                <Flex
                    className="error-boundary-stacks-row"
                    flexWrap={{ default: 'nowrap' }}
                    grow={{ default: 'grow' }}
                    shrink={{ default: 'shrink' }}
                    spaceItems={{ default: 'spaceItemsXl' }}
                >
                    <FlexItem className="error-boundary-stack" flex={{ default: 'flex_1' }}>
                        <Flex
                            className="error-boundary-stack-column"
                            direction={{ default: 'column' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title headingLevel="h2">Error stack</Title>
                            <FlexItem
                                className="error-boundary-stack-item"
                                flex={{ default: 'flex_1' }}
                            >
                                <ErrorBoundaryCodeBlock
                                    code={error.stack ?? ''}
                                    idForButton="error-boundary-button-error-stack"
                                    idForContent="error-boundary-content-error-stack"
                                    phraseForCopied="Copied to clipboard: Error stack"
                                    phraseForCopy="Copy to clipboard: Error stack"
                                />
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    <FlexItem className="error-boundary-stack" flex={{ default: 'flex_1' }}>
                        <Flex
                            className="error-boundary-stack-column"
                            direction={{ default: 'column' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <Title headingLevel="h2">Component stack</Title>
                            <FlexItem
                                className="error-boundary-stack-item"
                                flex={{ default: 'flex_1' }}
                            >
                                <ErrorBoundaryCodeBlock
                                    code={errorInfo.componentStack}
                                    idForButton="error-boundary-button-component-stack"
                                    idForContent="error-boundary-content-component-stack"
                                    phraseForCopied="Copied to clipboard: Component stack"
                                    phraseForCopy="Copy to clipboard: Component stack"
                                />
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                </Flex>
            </Flex>
        </PageSection>
    );
}

export default ErrorBoundaryPage;
