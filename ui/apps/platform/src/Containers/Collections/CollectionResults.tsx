import React, { useCallback, useEffect, useState, ReactNode } from 'react';
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
    Skeleton,
    Text,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, ListIcon, SyncAltIcon } from '@patternfly/react-icons';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';

import { CollectionRequest, dryRunCollection } from 'services/CollectionsService';
import { ListDeployment } from 'types/deployment.proto';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import { CollectionConfigError, parseConfigError } from './errorUtils';
import { SelectorEntityType } from './types';

function fetchMatchingDeployments(
    dryRunConfig: CollectionRequest,
    page: number,
    filterText: string,
    entity: SelectorEntityType
) {
    const pageSize = 10;
    const query = { [entity]: filterText };
    const sortOption = {
        field: 'Deployment',
        reversed: false,
    };
    const { request } = dryRunCollection(dryRunConfig, query, page, pageSize, sortOption);
    return request;
}

function DeploymentSkeleton() {
    return (
        <Flex className="pf-u-mb-0">
            <FlexItem style={{ flex: '0 1 24px' }}>
                <Skeleton />
            </FlexItem>
            <FlexItem className="pf-u-flex-grow-1">
                <Skeleton className="pf-u-mb-sm" fontSize="sm" />
                <Skeleton fontSize="sm" />
            </FlexItem>
            <Divider component="div" className="pf-u-mt-xs" />
        </Flex>
    );
}

function DeploymentResult({ deployment }: { deployment: ListDeployment }) {
    return (
        <Flex>
            <FlexItem>
                <ResourceIcon kind="Deployment" />
            </FlexItem>
            <FlexItem>
                <div>{deployment.name}</div>
                <span className="pf-u-color-300 pf-u-font-size-xs">
                    In &quot;{deployment.cluster} / {deployment.namespace}
                    &quot;
                </span>
            </FlexItem>
            <Divider className="pf-u-mt-md" />
        </Flex>
    );
}

export type CollectionResultsProps = {
    headerContent?: ReactNode;
    dryRunConfig: CollectionRequest;
    configError?: CollectionConfigError;
    setConfigError?: (newError: CollectionConfigError | undefined) => void;
};

function CollectionResults({
    headerContent,
    dryRunConfig,
    configError,
    setConfigError,
}: CollectionResultsProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [selected, setSelected] = useState<SelectorEntityType>('Deployment');
    // This state controls the value of the text in the SearchInput component separately from the value sent via query
    const [filterInput, setFilterInput] = useState('');
    // This state controls the filter value that we want to use for queries, and is set by manual user interaction.
    const [filterValue, setFilterValue] = useState('');
    const queryFn = useCallback(
        (page: number) => fetchMatchingDeployments(dryRunConfig, page, filterValue, selected),
        [dryRunConfig, filterValue, selected]
    );
    const {
        data,
        fetchNextPage,
        resetPages,
        clearPages,
        isEndOfResults,
        isFetchingNextPage,
        isRefreshingResults,
    } = usePaginatedQuery(queryFn, 10, {
        debounceRate: 800,
        manualFetch: true,
        dedupKeyFn: ({ id }) => id,
        onError: (err) => {
            setConfigError?.(parseConfigError(err));
        },
    });

    const selectorRulesExist =
        dryRunConfig.resourceSelectors?.[0]?.rules?.length > 0 ||
        dryRunConfig.embeddedCollectionIds.length > 0;

    function onRuleOptionSelect(_, value): void {
        setSelected(value);
        setFilterInput('');
        setFilterValue('');
        closeSelect();
    }

    function onSearchInputChange(_event, value) {
        setFilterInput(value);
    }

    useEffect(() => {
        if (configError) {
            clearPages();
        }
    }, [clearPages, configError]);

    const refreshResults = useCallback(() => {
        setConfigError?.(undefined);
        if (selectorRulesExist) {
            resetPages();
        }
    }, [resetPages, selectorRulesExist, setConfigError]);

    useEffect(() => {
        refreshResults();
    }, [refreshResults]);

    let content: ReactNode = '';

    if (configError) {
        content = (
            <EmptyState variant={EmptyStateVariant.xs}>
                <EmptyStateIcon
                    style={{ color: 'var(--pf-global--danger-color--200)' }}
                    icon={ExclamationCircleIcon}
                />
                <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
                    <Title headingLevel="h2" size="md">
                        {configError.message}
                    </Title>
                    <p>{configError.details}</p>
                </Flex>
            </EmptyState>
        );
    } else if (!selectorRulesExist) {
        content = (
            <EmptyState variant={EmptyStateVariant.xs}>
                <EmptyStateIcon icon={ListIcon} />
                <p>Add selector rules or attach existing collections to view resource matches</p>
            </EmptyState>
        );
    } else {
        content = (
            <Flex
                direction={{ default: 'column' }}
                grow={{ default: 'grow' }}
                className="pf-u-mt-lg"
            >
                {isRefreshingResults ? (
                    <>
                        {Array.from(Array(10).keys()).map((index: number) => (
                            <DeploymentSkeleton key={`refreshing-deployment-${index}`} />
                        ))}
                    </>
                ) : (
                    <>
                        {data.map((page) =>
                            page.map((deployment: ListDeployment) => (
                                <DeploymentResult key={deployment.id} deployment={deployment} />
                            ))
                        )}
                        {!isEndOfResults ? (
                            <Button
                                variant="link"
                                isInline
                                className="pf-u-text-align-center"
                                isLoading={isFetchingNextPage}
                                onClick={() => fetchNextPage(true)}
                            >
                                View more
                            </Button>
                        ) : (
                            <span className="pf-u-color-300 pf-u-text-align-center pf-u-font-size-sm">
                                end of results
                            </span>
                        )}
                    </>
                )}
            </Flex>
        );
    }

    return (
        <>
            <div className="pf-u-p-lg pf-u-display-flex pf-u-align-items-center">
                <div className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1">
                    <Title headingLevel="h2">Collection results</Title>
                    <Text>See a preview of current matches.</Text>
                </div>
                <Button
                    variant="plain"
                    onClick={refreshResults}
                    title="Refresh results"
                    isDisabled={isRefreshingResults}
                >
                    <SyncAltIcon />
                </Button>
                {headerContent}
            </div>
            <Divider />
            <div className="pf-u-h-100 pf-u-p-lg" style={{ overflow: 'auto' }}>
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    alignItems={{ default: 'alignItemsStretch' }}
                    direction={{ default: 'column' }}
                >
                    <Flex spaceItems={{ default: 'spaceItemsNone' }}>
                        <FlexItem>
                            <Select
                                toggleAriaLabel="Select an entity type to filter the results by"
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
                        <div className="pf-u-flex-grow-1 pf-u-flex-basis-0">
                            <SearchInput
                                aria-label="Filter by name"
                                placeholder="Filter by name"
                                value={filterInput}
                                onChange={onSearchInputChange}
                                onSearch={() => setFilterValue(filterInput)}
                                onClear={() => {
                                    setFilterInput('');
                                    setFilterValue('');
                                }}
                            />
                        </div>
                    </Flex>
                    {content}
                </Flex>
            </div>
        </>
    );
}

export default CollectionResults;
