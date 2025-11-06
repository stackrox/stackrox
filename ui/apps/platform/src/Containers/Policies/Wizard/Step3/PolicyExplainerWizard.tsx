import { useEffect, useState, useRef } from 'react';
import type { ReactElement, ReactNode } from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Spinner,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import type { ClientPolicy, PolicySection } from 'types/policy.proto';
import { generatePolicyExplanation } from 'services/VertexAIService';

const DEBOUNCE_DELAY_MS = 2000; // 2 seconds delay to avoid excessive API calls

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

/**
 * Validates if policy criteria are complete (all added fields have at least one value configured)
 * Note: Some fields have optional sub-fields, so we only check if the field has ANY non-empty value
 */
function validatePolicyCriteria(policySections: PolicySection[] | undefined): {
    isValid: boolean;
    hasContent: boolean;
    incompleteFields: string[];
} {
    if (!policySections || policySections.length === 0) {
        return { isValid: true, hasContent: false, incompleteFields: [] };
    }

    const incompleteFields: string[] = [];
    let hasAnyGroups = false;

    for (const section of policySections) {
        if (section.policyGroups && section.policyGroups.length > 0) {
            hasAnyGroups = true;
            for (const group of section.policyGroups) {
                // Only mark as incomplete if the field has NO values at all
                // or if ALL values are empty/whitespace (this handles mandatory fields)
                if (!group.values || group.values.length === 0) {
                    incompleteFields.push(group.fieldName);
                } else {
                    // Check if there's at least one non-empty value
                    // Note: Fields can have data in 'key', 'value', or 'arrayValue'
                    // - 'key' + 'value': e.g., Image Component (key=componentName, value=version)
                    // - 'value': e.g., Image Registry (value=registry name)
                    // - 'arrayValue': e.g., Image Signature Verified By (arrayValue=[...])
                    // TypeScript type only defines 'value', so we use type assertion for others
                    const allValuesEmpty = group.values.every((v) => {
                        const vAny = v as any;
                        const keyEmpty = !vAny.key || (typeof vAny.key === 'string' && vAny.key.trim() === '');
                        const valueEmpty = !v.value || v.value.trim() === '';
                        const arrayValueEmpty = !vAny.arrayValue || (Array.isArray(vAny.arrayValue) && vAny.arrayValue.length === 0);
                        return keyEmpty && valueEmpty && arrayValueEmpty;
                    });
                    
                    if (allValuesEmpty) {
                        incompleteFields.push(group.fieldName);
                    }
                }
            }
        }
    }

    return {
        isValid: incompleteFields.length === 0,
        hasContent: hasAnyGroups,
        incompleteFields,
    };
}

/**
 * Wizard-specific wrapper for PolicyExplainer that reads from Formik context
 * and implements debouncing to avoid excessive API calls while user is editing
 */
function PolicyExplainerWizard(): ReactElement {
    const { values } = useFormikContext<ClientPolicy>();
    const [explanation, setExplanation] = useState<string>('');
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);
    const [showFullText, setShowFullText] = useState<boolean>(false);
    const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);
    const lastPolicyRef = useRef<string>('');

    useEffect(() => {
        // Serialize the policy criteria to detect changes
        const policyCriteria = JSON.stringify({
            name: values.name,
            description: values.description,
            rationale: values.rationale,
            severity: values.severity,
            categories: values.categories,
            lifecycleStages: values.lifecycleStages,
            enforcementActions: values.enforcementActions,
            policySections: values.policySections,
            scope: values.scope,
            exclusions: values.exclusions,
        });

        // Only trigger if policy criteria actually changed
        if (policyCriteria === lastPolicyRef.current) {
            return;
        }

        lastPolicyRef.current = policyCriteria;

        // Clear any existing timer
        if (debounceTimerRef.current) {
            clearTimeout(debounceTimerRef.current);
        }

        // Validate policy criteria
        const validation = validatePolicyCriteria(values.policySections);

        // Don't fetch if there are no policy sections yet
        if (!validation.hasContent) {
            setExplanation('');
            setIsLoading(false);
            setError(null);
            return;
        }

        // Don't fetch if criteria are incomplete
        if (!validation.isValid) {
            setExplanation('');
            setIsLoading(false);
            setError(null);
            return;
        }

        // Set loading state immediately
        setIsLoading(true);
        setError(null);

        // Debounce the API call
        debounceTimerRef.current = setTimeout(async () => {
            try {
                const result = await generatePolicyExplanation(values);
                setExplanation(result);
                setIsLoading(false);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to generate explanation');
                setIsLoading(false);
            }
        }, DEBOUNCE_DELAY_MS);

        // Cleanup function
        return () => {
            if (debounceTimerRef.current) {
                clearTimeout(debounceTimerRef.current);
            }
        };
    }, [
        values.name,
        values.description,
        values.rationale,
        values.severity,
        values.categories,
        values.lifecycleStages,
        values.enforcementActions,
        values.policySections,
        values.scope,
        values.exclusions,
    ]);

    // Check validation status for display
    const validation = validatePolicyCriteria(values.policySections);

    // Determine card content based on state
    let cardContent: ReactElement;

    if (!validation.hasContent) {
        cardContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px' }}>
                Add policy criteria to see an AI-generated explanation of when this policy will
                trigger.
            </div>
        );
    } else if (!validation.isValid) {
        const fieldList = validation.incompleteFields.map((field) => `"${field}"`).join(', ');
        cardContent = (
            <Alert variant="warning" title="Incomplete policy criteria" isInline>
                The following field(s) have been added but not configured: {fieldList}. Please
                complete the configuration to generate an explanation.
            </Alert>
        );
    } else if (isLoading) {
        cardContent = (
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
        );
    } else if (error) {
        cardContent = (
            <Alert variant="danger" title="Failed to generate explanation" isInline>
                {error}
            </Alert>
        );
    } else if (explanation) {
        const lines = explanation.split('\n');
        const shouldTruncate = lines.length > 3;
        
        cardContent = (
            <>
                <div
                    style={{
                        whiteSpace: 'pre-wrap',
                        lineHeight: '1.7',
                        fontSize: '14px',
                        fontFamily: 'var(--pf-v5-global--FontFamily--text)',
                        padding: '8px 0',
                        color: 'var(--pf-v5-global--Color--100)',
                        ...(shouldTruncate && !showFullText
                            ? {
                                  overflow: 'hidden',
                                  display: '-webkit-box',
                                  WebkitLineClamp: 3,
                                  WebkitBoxOrient: 'vertical',
                              }
                            : {}),
                    }}
                >
                    {formatText(explanation)}
                </div>
                {shouldTruncate && (
                    <Button
                        variant="link"
                        isInline
                        onClick={() => setShowFullText(!showFullText)}
                        style={{ padding: '8px 0 0 0' }}
                    >
                        {showFullText ? 'Show less' : 'Show more'}
                    </Button>
                )}
            </>
        );
    } else {
        cardContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px' }}>
                Waiting for policy criteria to be defined...
            </div>
        );
    }

    return (
        <Card isFlat>
            <CardHeader>
                <CardTitle>When This Policy Triggers</CardTitle>
            </CardHeader>
            <CardBody style={{ maxHeight: '400px', overflowY: 'auto' }}>
                {cardContent}
            </CardBody>
        </Card>
    );
}

export default PolicyExplainerWizard;

