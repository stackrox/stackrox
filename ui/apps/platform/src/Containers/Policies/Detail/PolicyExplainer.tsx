import { useEffect, useState } from 'react';
import type { ReactElement, ReactNode } from 'react';
import {
    Alert,
    Card,
    CardBody,
    CardHeader,
    ExpandableSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import type { BasePolicy } from 'types/policy.proto';
import { generatePolicyExplanation } from 'services/VertexAIService';

// Parse simple formatting markers (**bold**) and apply PatternFly styling
function formatText(text: string): ReactNode[] {
    const lines = text.split('\n');
    return lines.map((line, lineIndex) => {
        // Parse **bold** markers
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
                <strong key={`${lineIndex}-${match.index}`} style={{ fontWeight: 600 }}>
                    {match[1]}
                </strong>
            );
            lastIndex = match.index + match[0].length;
        }

        // Add remaining text after last bold marker
        if (lastIndex < line.length) {
            parts.push(line.substring(lastIndex));
        }

        // Return line with a newline
        return (
            <span key={lineIndex}>
                {parts.length > 0 ? parts : line}
                {lineIndex < lines.length - 1 && '\n'}
            </span>
        );
    });
}

type PolicyExplainerProps = {
    policy: BasePolicy;
};

function PolicyExplainer({ policy }: PolicyExplainerProps): ReactElement {
    const [explanation, setExplanation] = useState<string>('');
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [isExpanded, setIsExpanded] = useState<boolean>(true);

    useEffect(() => {
        let isMounted = true;

        async function fetchExplanation() {
            setIsLoading(true);
            setError(null);

            try {
                const result = await generatePolicyExplanation(policy);
                if (isMounted) {
                    setExplanation(result);
                    setIsLoading(false);
                }
            } catch (err) {
                if (isMounted) {
                    setError(err instanceof Error ? err.message : 'Failed to generate explanation');
                    setIsLoading(false);
                }
            }
        }

        fetchExplanation();

        return () => {
            isMounted = false;
        };
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
                    <Alert variant="danger" title="Failed to generate explanation" isInline>
                        {error}
                    </Alert>
                )}

                {!isLoading && !error && explanation && (
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
                )}
            </CardBody>
        </Card>
    );
}

export default PolicyExplainer;

