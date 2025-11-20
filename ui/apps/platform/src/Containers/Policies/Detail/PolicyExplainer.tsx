import { useEffect, useState } from 'react';
import type { ReactElement, ReactNode } from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    CardHeader,
    ExpandableSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { SyncAltIcon } from '@patternfly/react-icons';

import type { BasePolicy } from 'types/policy.proto';
import { generatePolicyExplanation } from 'services/VertexAIService';

// Parse formatting markers (**bold**) and detect tables for proper monospace rendering
function formatText(text: string): ReactNode[] {
    const lines = text.split('\n');
    const result: ReactNode[] = [];
    let i = 0;

    while (i < lines.length) {
        const line = lines[i];

        // Detect table rows (lines starting with |)
        if (line.trim().startsWith('|')) {
            // Collect all consecutive table lines
            const tableLines: string[] = [];
            while (i < lines.length && lines[i].trim().startsWith('|')) {
                tableLines.push(lines[i]);
                i++;
            }

            // Render table as preformatted block
            result.push(
                <pre
                    key={`table-${i}`}
                    style={{
                        fontFamily: 'var(--pf-v5-global--FontFamily--monospace)',
                        fontSize: '13px',
                        lineHeight: '1.6',
                        margin: '12px 0',
                        padding: '12px',
                        backgroundColor: 'var(--pf-v5-global--BackgroundColor--200)',
                        border: '1px solid var(--pf-v5-global--BorderColor--100)',
                        borderRadius: '3px',
                        overflowX: 'auto',
                        whiteSpace: 'pre',
                    }}
                >
                    {tableLines.join('\n')}
                </pre>
            );
        } else {
            // Regular line - parse **bold** markers
            const parts: ReactNode[] = [];
            let lastIndex = 0;
            const boldRegex = /\*\*(.+?)\*\*/g;
            let match;

            while ((match = boldRegex.exec(line)) !== null) {
                // Add text before the bold marker
                if (match.index > lastIndex) {
                    parts.push(line.substring(lastIndex, match.index));
                }
                // Add bolded text
                parts.push(
                    <strong key={`${i}-${match.index}`} style={{ fontWeight: 600 }}>
                        {match[1]}
                    </strong>
                );
                lastIndex = match.index + match[0].length;
            }

            // Add remaining text after last bold marker
            if (lastIndex < line.length) {
                parts.push(line.substring(lastIndex));
            }

            // Add line with newline
            result.push(
                <span key={i}>
                    {parts.length > 0 ? parts : line}
                    {i < lines.length - 1 && '\n'}
                </span>
            );
            i++;
        }
    }

    return result;
}

type PolicyExplainerProps = {
    policy: BasePolicy;
};

function PolicyExplainer({ policy }: PolicyExplainerProps): ReactElement {
    const [explanation, setExplanation] = useState<string>('');
    const [warning, setWarning] = useState<string | undefined>(undefined);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [isExpanded, setIsExpanded] = useState<boolean>(true);

    const fetchExplanation = async () => {
        setIsLoading(true);
        setError(null);

        try {
            const result = await generatePolicyExplanation(policy);
            setExplanation(result.explanation);
            setWarning(result.warning);
            setIsLoading(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to generate explanation');
            setIsLoading(false);
        }
    };

    // Function to manually regenerate the explanation
    const handleRegenerate = () => {
        fetchExplanation();
    };

    useEffect(() => {
        fetchExplanation();
    }, [policy]);

    const onToggle = (expanded: boolean) => {
        setIsExpanded(expanded);
    };

    return (
        <Card isFlat>
            <CardHeader>
                <Title headingLevel="h3">When This Policy Triggers</Title>
            </CardHeader>
            <CardBody>
                {isLoading && (
                    <div
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '12px',
                            padding: '20px 0',
                        }}
                    >
                        <Spinner size="md" />
                        <span>Generating explanation...</span>
                    </div>
                )}

                {error && (
                    <>
                        <Alert variant="danger" title="Failed to generate explanation" isInline>
                            {error}
                        </Alert>
                        <Button
                            variant="link"
                            icon={<SyncAltIcon />}
                            onClick={handleRegenerate}
                            style={{ padding: '12px 0 0 0' }}
                        >
                            Regenerate
                        </Button>
                    </>
                )}

                {!isLoading && !error && explanation && (
                    <>
                        {warning && (
                            <Alert
                                variant="warning"
                                title="Policy Sanity Check"
                                isInline
                                style={{ marginBottom: '16px' }}
                            >
                                {warning}
                            </Alert>
                        )}
                        <div
                            style={{
                                whiteSpace: 'pre-wrap',
                                lineHeight: '1.7',
                                fontSize: '14px',
                                fontFamily: 'var(--pf-v5-global--FontFamily--text)',
                                padding: '8px 0',
                                color: 'var(--pf-v5-global--Color--100)',
                            }}
                        >
                            {formatText(explanation)}
                        </div>
                        <Button
                            variant="link"
                            icon={<SyncAltIcon />}
                            onClick={handleRegenerate}
                            style={{ padding: '8px 0 0 0' }}
                        >
                            Regenerate
                        </Button>
                    </>
                )}
            </CardBody>
        </Card>
    );
}

export default PolicyExplainer;

