import React, { useCallback, useEffect, useState, useMemo } from 'react';
import {
    Button,
    Divider,
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    Flex,
    FlexItem,
    Select,
    SearchInput,
    SelectOption,
    debounce,
} from '@patternfly/react-core';
import { ListIcon } from '@patternfly/react-icons';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';

import { dryRunCollection } from 'services/CollectionsService';
import { ListDeployment } from 'types/deployment.proto';

function CollectionResults({ dryRunConfig }) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [selected, setSelected] = useState('Deployment');
    const [filterText, setFilterText] = useState('');
    const [isEndOfResults, setIsEndOfResults] = useState(false);
    const [deployments, setDeployments] = useState<ListDeployment[]>([]);
    const [hasRuleSelected, setHasRuleSelected] = useState(false);

    function onRuleOptionSelect(_, value) {
        setSelected(value);
        setFilterText('');
        closeSelect();
    }

    const fetchDryRun = useCallback(
        (page: number) => {
            const pageSize = 10;
            const query = filterText ? `${selected}:${filterText}` : '';
            const { request } = dryRunCollection(dryRunConfig, query, page, pageSize);
            request
                .then((results) => {
                    setIsEndOfResults(results.length < 10);
                    setDeployments((current) => (page === 0 ? results : [...current, ...results]));
                    setHasRuleSelected(true);
                })
                .catch(() => {
                    // TODO: indicate results not loading properly?
                });
        },
        [dryRunConfig, filterText, selected]
    );

    useEffect(() => {
        if (dryRunConfig.resourceSelectors?.[0]?.rules?.length > 0) {
            fetchDryRun(0);
        } else {
            setHasRuleSelected(false);
        }
    }, [dryRunConfig, fetchDryRun]);

    const onSearchInputChange = useMemo(
        () =>
            debounce((value: string) => {
                setFilterText(value);
            }, 800),
        []
    );

    const currentPage = Math.floor((deployments.length - 1) / 10);

    return (
        <>
            <Divider />
            {!hasRuleSelected ? (
                <EmptyState variant={EmptyStateVariant.xs}>
                    <EmptyStateIcon icon={ListIcon} />
                    <p>
                        Add selector rules or attach existing collections to view resource matches
                    </p>
                </EmptyState>
            ) : (
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-mt-lg"
                >
                    <FlexItem>
                        <Select
                            toggleAriaLabel="Select by name or label"
                            isOpen={isOpen}
                            onToggle={onToggle}
                            selections={selected}
                            onSelect={onRuleOptionSelect}
                            isDisabled={false}
                        >
                            <SelectOption value="Deployment">Deployment</SelectOption>
                            <SelectOption value="Namespace">Namespace</SelectOption>
                            <SelectOption value="Cluster">Cluster</SelectOption>
                        </Select>
                    </FlexItem>
                    <FlexItem grow={{ default: 'grow' }}>
                        <SearchInput
                            aria-label="Filter by name"
                            placeholder="Filter by name"
                            value={filterText}
                            onChange={onSearchInputChange}
                        />
                    </FlexItem>
                    <Flex
                        direction={{ default: 'column' }}
                        grow={{ default: 'grow' }}
                        className="pf-u-mt-lg"
                    >
                        {deployments.map((deployment) => {
                            return (
                                <Flex key={deployment.id}>
                                    <FlexItem>
                                        <ResourceIcon kind="Deployment" />
                                    </FlexItem>
                                    <FlexItem>
                                        <div>{deployment.name}</div>
                                        <span className="pf-u-color-400 pf-u-font-size-xs">
                                            In &quot;{deployment.cluster} / {deployment.namespace}
                                            &quot;
                                        </span>
                                    </FlexItem>
                                    <Divider className="pf-u-mt-md" />
                                </Flex>
                            );
                        })}
                        {!isEndOfResults ? (
                            <Button
                                variant="link"
                                isInline
                                className="pf-u-text-align-center"
                                onClick={() => fetchDryRun(currentPage + 1)}
                            >
                                View more
                            </Button>
                        ) : (
                            <span className="pf-u-color-400 pf-u-text-align-center pf-u-font-size-sm">
                                end of results
                            </span>
                        )}
                    </Flex>
                </Flex>
            )}
        </>
    );
}

export default CollectionResults;
