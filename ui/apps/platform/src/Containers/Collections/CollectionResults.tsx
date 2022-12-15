import React, { useCallback, useEffect, useState, useMemo, ReactNode } from 'react';
import {
    debounce,
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
    Spinner,
    Text,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, ListIcon, SyncAltIcon } from '@patternfly/react-icons';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';

import { CollectionRequest, dryRunCollection } from 'services/CollectionsService';
import { ListDeployment } from 'types/deployment.proto';
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
    const { request } = dryRunCollection(dryRunConfig, query, page, pageSize);
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
                <span className="pf-u-color-400 pf-u-font-size-xs">
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
    setConfigError = () => {},
}: CollectionResultsProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [isRefreshingResults, setIsRefreshingResults] = useState(false);
    const [selected, setSelected] = useState<SelectorEntityType>('Deployment');
    const [filterText, setFilterText] = useState<string>('');
    const [isEndOfResults, setIsEndOfResults] = useState<boolean>(false);
    const [deployments, setDeployments] = useState<ListDeployment[]>([]);

    const currentPage: number = Math.floor((deployments.length - 1) / 10);
    const selectorRulesExist =
        dryRunConfig.resourceSelectors?.[0]?.rules?.length > 0 ||
        dryRunConfig.embeddedCollectionIds.length > 0;

    function onRuleOptionSelect(_, value): void {
        setSelected(value);
        setFilterText('');
        closeSelect();
    }

    const fetchDryRun = useCallback(
        (currConfig, currPage, currFilter, currEntity) => {
            fetchMatchingDeployments(currConfig, currPage, currFilter, currEntity)
                .then((results) => {
                    setIsEndOfResults(results.length < 10);
                    setDeployments((current) =>
                        currPage === 0 ? results : [...current, ...results]
                    );
                })
                .catch((err) => {
                    setConfigError(parseConfigError(err));
                })
                .finally(() => {
                    setIsRefreshingResults(false);
                });
        },
        [setConfigError]
    );

    const fetchDryRunDebounced = useMemo(() => debounce(fetchDryRun, 800), [fetchDryRun]);

    useEffect(() => {
        if (configError) {
            setDeployments([]);
        }
    }, [configError]);

    const refreshResults = useCallback(() => {
        setConfigError(undefined);
        if (selectorRulesExist) {
            setIsRefreshingResults(true);
            fetchDryRunDebounced(dryRunConfig, 0, filterText, selected);
        }
    }, [
        dryRunConfig,
        fetchDryRunDebounced,
        filterText,
        selected,
        selectorRulesExist,
        setConfigError,
    ]);

    useEffect(() => {
        refreshResults();
    }, [refreshResults]);

    let content: ReactNode = '';

    if (configError) {
        content = (
            <Flex className="pf-u-h-100" alignContent={{ default: 'alignContentCenter' }}>
                <EmptyState variant={EmptyStateVariant.xs}>
                    <EmptyStateIcon
                        style={{ color: 'var(--pf-global--danger-color--200)' }}
                        icon={ExclamationCircleIcon}
                    />
                    <Flex
                        spaceItems={{ default: 'spaceItemsMd' }}
                        direction={{ default: 'column' }}
                    >
                        <Title headingLevel="h2" size="md">
                            {configError.message}
                        </Title>
                        <p className="pf-u-text-align-left">{configError.details}</p>
                    </Flex>
                </EmptyState>
            </Flex>
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
                    <FlexItem grow={{ default: 'grow' }}>
                        <SearchInput
                            aria-label="Filter by name"
                            placeholder="Filter by name"
                            value={filterText}
                            onChange={setFilterText}
                        />
                    </FlexItem>
                </Flex>
                <Flex
                    direction={{ default: 'column' }}
                    grow={{ default: 'grow' }}
                    className="pf-u-mt-lg"
                >
                    {isRefreshingResults ? (
                        <>
                            {deployments.map((deployment: ListDeployment) => (
                                <DeploymentSkeleton key={`refreshing-${deployment.id}`} />
                            ))}
                            <Spinner className="pf-u-align-self-center" size="lg" />
                        </>
                    ) : (
                        <>
                            {deployments.map((deployment: ListDeployment) => (
                                <DeploymentResult key={deployment.id} deployment={deployment} />
                            ))}
                            {!isEndOfResults ? (
                                <Button
                                    variant="link"
                                    isInline
                                    className="pf-u-text-align-center"
                                    onClick={() => {
                                        fetchDryRun(
                                            dryRunConfig,
                                            currentPage + 1,
                                            filterText,
                                            selected
                                        );
                                    }}
                                >
                                    View more
                                </Button>
                            ) : (
                                <span className="pf-u-color-400 pf-u-text-align-center pf-u-font-size-sm">
                                    end of results
                                </span>
                            )}
                        </>
                    )}
                </Flex>
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
                {content}
            </div>
        </>
    );
}

export default CollectionResults;
