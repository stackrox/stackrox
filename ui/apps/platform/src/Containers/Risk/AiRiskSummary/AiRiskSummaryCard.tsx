import type { ReactElement } from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Content,
    Flex,
    FlexItem,
    Spinner,
} from '@patternfly/react-core';
import { CopyIcon, MagicIcon, TimesIcon } from '@patternfly/react-icons';

import useClipboardCopy from 'hooks/useClipboardCopy';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type AiRiskSummaryCardProps = {
    summary: string | undefined;
    isLoading: boolean;
    error: unknown;
    onClose: () => void;
};

function AiRiskSummaryCard({
    summary,
    isLoading,
    error,
    onClose,
}: AiRiskSummaryCardProps): ReactElement {
    const { wasCopied, copyToClipboard } = useClipboardCopy();

    return (
        <Card>
            <CardHeader
                actions={{
                    actions: (
                        <>
                            <Button
                                variant="plain"
                                icon={<CopyIcon />}
                                aria-label={
                                    wasCopied ? 'Copied AI summary' : 'Copy AI summary to clipboard'
                                }
                                isDisabled={!summary}
                                onClick={() => summary && copyToClipboard(summary)}
                            />
                            <Button
                                variant="plain"
                                icon={<TimesIcon />}
                                aria-label="Close AI investigation"
                                onClick={onClose}
                            />
                        </>
                    ),
                }}
            >
                <CardTitle>
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <MagicIcon />
                        <FlexItem>AI risk briefing</FlexItem>
                    </Flex>
                </CardTitle>
            </CardHeader>
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
        </Card>
    );
}

export default AiRiskSummaryCard;
