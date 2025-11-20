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
    Tabs,
    Tab,
    TabTitleText,
} from '@patternfly/react-core';
import { SyncAltIcon } from '@patternfly/react-icons';
import { useFormikContext } from 'formik';

import type { ClientPolicy, PolicySection } from 'types/policy.proto';
import { generatePolicyExplanation } from 'services/VertexAIService';
import PolicySimulator from './PolicySimulator';

const DEBOUNCE_DELAY_MS = 2000; // 2 seconds delay to avoid excessive API calls

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
                // Skip if group is malformed
                if (!group || !group.fieldName) {
                    continue;
                }
                
                // Only mark as incomplete if the field has NO values at all
                // or if ALL values are empty/whitespace (this handles mandatory fields)
                if (!group.values || !Array.isArray(group.values) || group.values.length === 0) {
                    incompleteFields.push(group.fieldName);
                } else {
                    // Check if there's at least one non-empty value
                    // Note: Fields can have data in 'key', 'value', or 'arrayValue'
                    // - 'key' + 'value': e.g., Image Component (key=componentName, value=version)
                    // - 'value': e.g., Image Registry (value=registry name)
                    // - 'arrayValue': e.g., Image Signature Verified By (arrayValue=[...])
                    // TypeScript type only defines 'value', so we use type assertion for others
                    const allValuesEmpty = group.values.every((v) => {
                        if (!v) return true;
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
 * Extracts the table portion from the AI explanation (lines starting with |)
 */
function extractTruthTable(text: string): string {
    const lines = text.split('\n');
    const tableLines: string[] = [];
    
    for (const line of lines) {
        if (line.trim().startsWith('|')) {
            tableLines.push(line);
        }
    }
    
    return tableLines.join('\n');
}

/**
 * Wizard-specific wrapper for PolicyExplainer that reads from Formik context
 * and implements debouncing to avoid excessive API calls while user is editing
 */
function PolicyExplainerWizard(): ReactElement {
    const { values } = useFormikContext<ClientPolicy>();
    const [explanation, setExplanation] = useState<string>('');
    const [warning, setWarning] = useState<string | undefined>(undefined);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);
    const [showFullText, setShowFullText] = useState<boolean>(false);
    const [activeTabKey, setActiveTabKey] = useState<string | number>('truthTable');
    const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);
    const lastPolicyRef = useRef<string>('');
    const previousSectionsRef = useRef<PolicySection[] | undefined>(undefined);

    // Function to manually regenerate the explanation (bypasses debounce)
    const handleRegenerate = async () => {
        // Clear any pending debounced calls
        if (debounceTimerRef.current) {
            clearTimeout(debounceTimerRef.current);
        }

        // Validate policy criteria
        const validation = validatePolicyCriteria(values.policySections);

        // Don't fetch if there are no policy sections yet or criteria are incomplete
        if (!validation.hasContent || !validation.isValid) {
            return;
        }

        // Set loading state
        setIsLoading(true);
        setError(null);

        try {
            const result = await generatePolicyExplanation(values);
            setExplanation(result.explanation);
            setWarning(result.warning);
            setIsLoading(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to generate explanation');
            setIsLoading(false);
        }
    };

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

        // Store previous sections for change detection in tables
        if (lastPolicyRef.current) {
            previousSectionsRef.current = JSON.parse(lastPolicyRef.current).policySections;
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
                setExplanation(result.explanation);
                setWarning(result.warning);
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

    // Extract truth table from explanation
    const truthTable = explanation ? extractTruthTable(explanation) : '';

    // Determine content for Truth Table tab
    let truthTableContent: ReactElement;
    
    if (!validation.hasContent) {
        truthTableContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px', padding: '16px' }}>
                Add policy criteria to see the truth table showing when this policy will trigger.
            </div>
        );
    } else if (!validation.isValid) {
        const fieldList = validation.incompleteFields.map((field) => `"${field}"`).join(', ');
        truthTableContent = (
            <Alert variant="warning" title="Incomplete policy criteria" isInline>
                The following field(s) have been added but not configured: {fieldList}. Please
                complete the configuration to generate the truth table.
            </Alert>
        );
    } else if (isLoading) {
        truthTableContent = (
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                    padding: '20px 0',
                }}
            >
                <Spinner size="md" />
                <span>Generating truth table...</span>
            </div>
        );
    } else if (error) {
        truthTableContent = (
            <>
                <Alert variant="danger" title="Failed to generate truth table" isInline>
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
        );
    } else if (truthTable) {
        truthTableContent = (
            <>
                <div
                    style={{
                        fontSize: '13px',
                        color: 'var(--pf-v5-global--Color--200)',
                        marginBottom: '12px',
                    }}
                >
                    This table shows all possible combinations of criteria and whether they would trigger a violation.
                </div>
                <pre
                    style={{
                        fontFamily: 'var(--pf-v5-global--FontFamily--monospace)',
                        fontSize: '13px',
                        lineHeight: '1.6',
                        margin: '0',
                        padding: '12px',
                        backgroundColor: 'var(--pf-v5-global--BackgroundColor--200)',
                        border: '1px solid var(--pf-v5-global--BorderColor--100)',
                        borderRadius: '3px',
                        overflowX: 'auto',
                        whiteSpace: 'pre',
                    }}
                >
                    {truthTable}
                </pre>
                <Button
                    variant="link"
                    icon={<SyncAltIcon />}
                    onClick={handleRegenerate}
                    style={{ padding: '12px 0 0 0' }}
                >
                    Regenerate
                </Button>
            </>
        );
    } else {
        truthTableContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px', padding: '16px' }}>
                Waiting for policy criteria to be defined...
            </div>
        );
    }

    // Determine content for AI explanation tab
    let aiExplanationContent: ReactElement;

    if (!validation.hasContent) {
        aiExplanationContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px', padding: '16px' }}>
                Add policy criteria to see an AI-generated explanation of when this policy will
                trigger.
            </div>
        );
    } else if (!validation.isValid) {
        const fieldList = validation.incompleteFields.map((field) => `"${field}"`).join(', ');
        aiExplanationContent = (
            <Alert variant="warning" title="Incomplete policy criteria" isInline>
                The following field(s) have been added but not configured: {fieldList}. Please
                complete the configuration to generate an explanation.
            </Alert>
        );
    } else if (isLoading) {
        aiExplanationContent = (
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
        aiExplanationContent = (
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
        );
    } else if (explanation) {
        const lines = explanation.split('\n');
        const shouldTruncate = lines.length > 3;
        
        aiExplanationContent = (
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
                <div style={{ display: 'flex', gap: '16px', padding: '8px 0 0 0' }}>
                    {shouldTruncate && (
                        <Button
                            variant="link"
                            isInline
                            onClick={() => setShowFullText(!showFullText)}
                        >
                            {showFullText ? 'Show less' : 'Show more'}
                        </Button>
                    )}
                    <Button
                        variant="link"
                        icon={<SyncAltIcon />}
                        onClick={handleRegenerate}
                        isInline
                    >
                        Regenerate
                    </Button>
                </div>
            </>
        );
    } else {
        aiExplanationContent = (
            <div style={{ color: 'var(--pf-v5-global--Color--200)', fontSize: '14px', padding: '16px' }}>
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
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(event, tabIndex) => setActiveTabKey(tabIndex)}
                    aria-label="Policy explanation tabs"
                >
                    <Tab 
                        eventKey="truthTable" 
                        title={<TabTitleText>Truth Table</TabTitleText>}
                        aria-label="Policy truth table"
                    >
                        <div style={{ padding: '16px 0' }}>
                            {truthTableContent}
                        </div>
                    </Tab>
                    <Tab 
                        eventKey="simulator" 
                        title={<TabTitleText>Interactive Simulator</TabTitleText>}
                        aria-label="Interactive deployment simulator"
                    >
                        <div style={{ padding: '16px 0' }}>
                            <PolicySimulator policySections={values.policySections} />
                        </div>
                    </Tab>
                    <Tab 
                        eventKey="ai" 
                        title={<TabTitleText>AI Explanation</TabTitleText>}
                        aria-label="AI generated explanation"
                    >
                        <div style={{ padding: '16px 0' }}>
                            {aiExplanationContent}
                        </div>
                    </Tab>
                </Tabs>
            </CardBody>
        </Card>
    );
}

export default PolicyExplainerWizard;

