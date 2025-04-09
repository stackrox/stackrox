import React, { useEffect, useMemo } from 'react';
import {
    Alert,
    AlertGroup,
    Bullseye,
    Button,
    Divider,
    EmptyState,
    Spinner,
    Stack,
    StackItem,
    EmptyStateHeader,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';

import download from 'utils/download';
import SelectSingle from 'Components/SelectSingle';
import useFetchNetworkPolicies from 'hooks/useFetchNetworkPolicies';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import CodeViewer from '../../../Components/CodeViewer';

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
                <Spinner size="lg" />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert
                isInline
                variant="danger"
                title={getAxiosErrorMessage(error)}
                component="p"
                className="pf-v5-u-mb-lg"
            />
        );
    }

    let policyErrorBanner: React.ReactNode = null;

    if (networkPolicyErrors.length > 0) {
        policyErrorBanner = (
            <AlertGroup className="pf-v5-u-mb-lg">
                {networkPolicyErrors.map((networkPolicyError) => (
                    <Alert
                        isInline
                        variant="warning"
                        title="There was an error loading network policy data"
                        component="p"
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
                    <EmptyState variant="xs">
                        <EmptyStateHeader titleText="No network policies" headingLevel="h4" />
                    </EmptyState>
                </Bullseye>
            </>
        );
    }

    return (
        <div className="pf-v5-u-h-100 pf-v5-u-p-md">
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
                {!!selectedNetworkPolicy?.yaml && (
                    <>
                        <StackItem>
                            <div className="pf-v5-u-h-100">
                                <CodeViewer code={selectedNetworkPolicy.yaml} />
                            </div>
                        </StackItem>
                        <StackItem>
                            <Button onClick={exportYAMLHandler}>Export YAML</Button>
                        </StackItem>
                    </>
                )}
            </Stack>
        </div>
    );
}

export default NetworkPolicies;
