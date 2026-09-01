import type { ReactElement } from 'react';
import {
    Alert,
    AlertActionLink,
    Bullseye,
    Button,
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    Content,
    Flex,
    FlexItem,
    Spinner,
} from '@patternfly/react-core';
import { CopyIcon } from '@patternfly/react-icons';

import useClipboardCopy from 'hooks/useClipboardCopy';
import AiExperienceIcon from 'images/aiExperience.svg?react';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type AiRiskSummaryCardProps = {
    summary: string | undefined;
    isLoading: boolean;
    error: unknown;
    isExpanded: boolean;
    onExpand: () => void;
    onRetry: () => void;
};

function AiRiskSummaryCard({
    summary,
    isLoading,
    error,
    isExpanded,
    onExpand,
    onRetry,
}: AiRiskSummaryCardProps): ReactElement {
    const { wasCopied, copyToClipboard } = useClipboardCopy();

    return (
        <Card id="ai-risk-summary-card" isExpanded={isExpanded}>
            <CardHeader
                onExpand={onExpand}
                toggleButtonProps={{
                    id: 'ai-risk-summary-toggle',
                    'aria-label': isExpanded
                        ? 'Collapse AI risk briefing'
                        : 'Expand AI risk briefing',
                    'aria-labelledby': 'ai-risk-summary-card ai-risk-summary-toggle',
                    'aria-expanded': isExpanded,
                }}
                actions={{
                    actions: (
                        <Button
                            variant="plain"
                            icon={<CopyIcon />}
                            aria-label={
                                wasCopied ? 'Copied AI summary' : 'Copy AI summary to clipboard'
                            }
                            isDisabled={!summary}
                            onClick={() => summary && copyToClipboard(summary)}
                        />
                    ),
                }}
            >
                <CardTitle>
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <AiExperienceIcon />
                        <FlexItem>AI risk briefing</FlexItem>
                    </Flex>
                </CardTitle>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    {isLoading && (
                        <Bullseye>
                            <Spinner aria-label="Generating AI risk summary" />
                        </Bullseye>
                    )}
                    {!isLoading && Boolean(error) && (
                        <Alert
                            variant="danger"
                            isInline
                            title="Unable to generate AI risk summary"
                            component="p"
                            actionLinks={
                                <AlertActionLink onClick={onRetry}>Try again</AlertActionLink>
                            }
                        >
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    )}
                    {!isLoading && !error && (
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            <Alert
                                variant="info"
                                isInline
                                title="Always review AI-generated content prior to use."
                                component="p"
                            />
                            <Content component="p" style={{ whiteSpace: 'pre-wrap' }}>
                                {summary}
                            </Content>
                        </Flex>
                    )}
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
}

export default AiRiskSummaryCard;
