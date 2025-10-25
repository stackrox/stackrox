import { useCallback, useEffect, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import {
    Button,
    Divider,
    EmptyState,
    EmptyStateFooter,
    EmptyStateHeader,
    EmptyStateIcon,
    Flex,
    FlexItem,
    SearchInput,
    SelectOption,
    Skeleton,
    Text,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, ListIcon, SyncAltIcon } from '@patternfly/react-icons';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

import { dryRunCollection } from 'services/CollectionsService';
import type { CollectionRequest } from 'services/CollectionsService';
import type { ListDeployment } from 'types/deployment.proto';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import { parseConfigError } from './errorUtils';
import type { CollectionConfigError } from './errorUtils';
import type { SelectorEntityType } from './types';

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
        <Flex className="pf-v5-u-mb-0">
            <FlexItem style={{ flex: '0 1 24px' }}>
                <Skeleton />
            </FlexItem>
            <FlexItem className="pf-v5-u-flex-grow-1">
                <Skeleton className="pf-v5-u-mb-sm" fontSize="sm" />
                <Skeleton fontSize="sm" />
            </FlexItem>
            <Divider component="div" className="pf-v5-u-mt-xs" />
        </Flex>
    );
}

function DeploymentResult({ deployment }: { deployment: ListDeployment }) {
    return (
        <>
            <Flex>
                <FlexItem>
                    <ResourceIcon kind="Deployment" />
                </FlexItem>
                <FlexItem>
                    <div>{deployment.name}</div>
                    <span className="pf-v5-u-color-300 pf-v5-u-font-size-xs">
                        In &quot;{deployment.cluster} / {deployment.namespace}
                        &quot;
                    </span>
                </FlexItem>
            </Flex>
            <Divider className="pf-v5-u-mt-sm" />
        </>
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

    function onRuleOptionSelect(_id: string, value: string): void {
        setSelected(value as SelectorEntityType);
        setFilterInput('');
        setFilterValue('');
    }

    function onSearchInputChange(_event: FormEvent<HTMLInputElement>, value: string) {
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
            <EmptyState variant="xs">
                <EmptyStateHeader
                    icon={
                        <EmptyStateIcon
                            style={{ color: 'var(--pf-v5-global--danger-color--200)' }}
                            icon={ExclamationCircleIcon}
                        />
                    }
                />
                <EmptyStateFooter>
                    <Flex
                        spaceItems={{ default: 'spaceItemsMd' }}
                        direction={{ default: 'column' }}
                    >
                        <Title headingLevel="h2" size="md">
                            {configError.message}
                        </Title>
                        <p>{configError.details}</p>
                    </Flex>
                </EmptyStateFooter>
            </EmptyState>
        );
    } else if (!selectorRulesExist) {
        content = (
            <EmptyState variant="xs">
                <EmptyStateHeader icon={<EmptyStateIcon icon={ListIcon} />} />
                <EmptyStateFooter>
                    <p>
                        Add selector rules or attach existing collections to view resource matches
                    </p>
                </EmptyStateFooter>
            </EmptyState>
        );
    } else {
        content = (
            <Flex
                direction={{ default: 'column' }}
                grow={{ default: 'grow' }}
                className="pf-v5-u-mt-lg"
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
                                className="pf-v5-u-text-align-center"
                                isLoading={isFetchingNextPage}
                                onClick={() => fetchNextPage(true)}
                            >
                                View more
                            </Button>
                        ) : (
                            <span className="pf-v5-u-color-300 pf-v5-u-text-align-center pf-v5-u-font-size-sm">
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
            <div className="pf-v5-u-p-lg pf-v5-u-display-flex pf-v5-u-align-items-center">
                <div className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1">
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
            <div className="pf-v5-u-h-100 pf-v5-u-p-lg" style={{ overflow: 'auto' }}>
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    alignItems={{ default: 'alignItemsStretch' }}
                    direction={{ default: 'column' }}
                >
                    <Flex spaceItems={{ default: 'spaceItemsNone' }}>
                        <FlexItem>
                            <SelectSingle
                                id="entity-type-select"
                                toggleAriaLabel="Select an entity type to filter the results by"
                                value={selected}
                                handleSelect={onRuleOptionSelect}
                                isDisabled={false}
                                isFullWidth={false}
                            >
                                <SelectOption value="Deployment">Deployment</SelectOption>
                                <SelectOption value="Namespace">Namespace</SelectOption>
                                <SelectOption value="Cluster">Cluster</SelectOption>
                            </SelectSingle>
                        </FlexItem>
                        <div className="pf-v5-u-flex-grow-1 pf-v5-u-flex-basis-0">
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
