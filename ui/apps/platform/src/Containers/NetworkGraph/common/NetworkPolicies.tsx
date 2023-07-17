import React, { useEffect, useMemo } from 'react';
import {
    Alert,
    AlertGroup,
    AlertVariant,
    Bullseye,
    Button,
    Divider,
    EmptyState,
    EmptyStateVariant,
    SelectOption,
    Spinner,
    Stack,
    StackItem,
    Title,
} from '@patternfly/react-core';
import { CodeEditor, Language } from '@patternfly/react-code-editor';

import download from 'utils/download';
import SelectSingle from 'Components/SelectSingle';
import { useTheme } from 'Containers/ThemeProvider';
import useFetchNetworkPolicies from 'hooks/useFetchNetworkPolicies';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import CodeEditorDarkModeControl from 'Components/PatternFly/CodeEditorDarkModeControl';

type NetworkPoliciesProps = {
    entityName: string;
    policyIds: string[];
};

type NetworkPolicyYAML = {
    name: string;
    yaml: string;
};

const allNetworkPoliciesId = 'All network policies';

function NetworkPolicies({ entityName, policyIds }: NetworkPoliciesProps): React.ReactElement {
    const { networkPolicies, networkPolicyErrors, isLoading, error } =
        useFetchNetworkPolicies(policyIds);
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    const allNetworkPoliciesYAML = useMemo(
        () => ({
            name: allNetworkPoliciesId,
            yaml: networkPolicies.map((networkPolicy) => networkPolicy.yaml).join('---\n'),
        }),
        [networkPolicies]
    );

    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = React.useState<
        NetworkPolicyYAML | undefined
    >(allNetworkPoliciesYAML);

    useEffect(() => {
        setSelectedNetworkPolicy(allNetworkPoliciesYAML);
    }, [allNetworkPoliciesYAML]);

    function onToggleDarkMode() {
        setCustomDarkMode((prevValue) => !prevValue);
    }

    function handleSelectedNetworkPolicy(_, value: string) {
        if (value !== allNetworkPoliciesId) {
            const newlySelectedNetworkPolicy = networkPolicies.find(
                (networkPolicy) => networkPolicy.name === value
            );
            setSelectedNetworkPolicy(newlySelectedNetworkPolicy);
        } else {
            setSelectedNetworkPolicy(allNetworkPoliciesYAML);
        }
    }

    function exportYAMLHandler() {
        if (selectedNetworkPolicy) {
            const fileName =
                selectedNetworkPolicy.name === allNetworkPoliciesId
                    ? entityName
                    : selectedNetworkPolicy.name;
            const fileContent = selectedNetworkPolicy.yaml;
            download(`${fileName}.yml`, fileContent, 'yml');
        }
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert
                isInline
                variant={AlertVariant.danger}
                title={getAxiosErrorMessage(error)}
                className="pf-u-mb-lg"
            />
        );
    }

    let policyErrorBanner: React.ReactNode = null;

    if (networkPolicyErrors.length > 0) {
        policyErrorBanner = (
            <AlertGroup className="pf-u-mb-lg">
                {networkPolicyErrors.map((networkPolicyError) => (
                    <Alert
                        isInline
                        variant={AlertVariant.warning}
                        title="There was an error loading network policy data"
                    >
                        {getAxiosErrorMessage(networkPolicyError)}
                    </Alert>
                ))}
            </AlertGroup>
        );
    }

    if (networkPolicies.length === 0) {
        return (
            <>
                {policyErrorBanner}
                <Bullseye>
                    <EmptyState variant={EmptyStateVariant.xs}>
                        <Title headingLevel="h4" size="md">
                            No network policies
                        </Title>
                    </EmptyState>
                </Bullseye>
            </>
        );
    }

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            {policyErrorBanner}
            <Stack hasGutter>
                <StackItem>
                    <SelectSingle
                        id="search-filter-attributes-select"
                        value={selectedNetworkPolicy?.name || ''}
                        handleSelect={handleSelectedNetworkPolicy}
                        placeholderText="Select a network policy"
                    >
                        <SelectOption value={allNetworkPoliciesId}>
                            All network policies
                        </SelectOption>
                        <Divider component="li" />
                        <>
                            {networkPolicies.map((networkPolicy) => {
                                return (
                                    <SelectOption
                                        key={networkPolicy.name}
                                        value={networkPolicy.name}
                                    >
                                        {networkPolicy.name}
                                    </SelectOption>
                                );
                            })}
                        </>
                    </SelectSingle>
                </StackItem>
                {selectedNetworkPolicy && (
                    <StackItem>
                        <div className="pf-u-h-100">
                            <CodeEditor
                                isDarkTheme={customDarkMode}
                                customControls={
                                    <CodeEditorDarkModeControl
                                        isDarkMode={customDarkMode}
                                        onToggleDarkMode={onToggleDarkMode}
                                    />
                                }
                                isCopyEnabled
                                isLineNumbersVisible
                                isReadOnly
                                code={selectedNetworkPolicy.yaml}
                                language={Language.yaml}
                                height="300px"
                            />
                        </div>
                    </StackItem>
                )}
                {selectedNetworkPolicy && (
                    <StackItem>
                        <Button onClick={exportYAMLHandler}>Export YAML</Button>
                    </StackItem>
                )}
            </Stack>
        </div>
    );
}

export default NetworkPolicies;
