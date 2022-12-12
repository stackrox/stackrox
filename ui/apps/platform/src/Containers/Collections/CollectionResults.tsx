import React, { useCallback, useEffect, useState, useMemo, ReactNode } from 'react';
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
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, ListIcon } from '@patternfly/react-icons';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';

import { CollectionRequest, dryRunCollection } from 'services/CollectionsService';
import { ListDeployment } from 'types/deployment.proto';
import { CollectionSaveError, parseSaveError } from './errorUtils';
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

export type CollectionResultsProps = {
    dryRunConfig: CollectionRequest;
    saveError?: CollectionSaveError;
    setSaveError?: (newError: CollectionSaveError | undefined) => void;
};

function CollectionResults({
    dryRunConfig,
    saveError,
    setSaveError = () => {},
}: CollectionResultsProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
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
                    setSaveError(parseSaveError(err));
                });
        },
        [setSaveError]
    );

    const fetchDryRunDebounced = useMemo(() => debounce(fetchDryRun, 800), [fetchDryRun]);

    useEffect(() => {
        if (saveError) {
            setDeployments([]);
        }
    }, [saveError]);

    useEffect(() => {
        setSaveError(undefined);
        if (selectorRulesExist) {
            fetchDryRunDebounced(dryRunConfig, 0, filterText, selected);
        }
    }, [
        dryRunConfig,
        fetchDryRunDebounced,
        filterText,
        selected,
        selectorRulesExist,
        setSaveError,
    ]);

    let content: ReactNode = '';

    if (saveError) {
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
                            {saveError.message}
                        </Title>
                        <p className="pf-u-text-align-left">{saveError.details}</p>
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
                alignItems={{ default: 'alignItemsCenter' }}
                className="pf-u-mt-lg"
            >
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
                <Flex
                    direction={{ default: 'column' }}
                    grow={{ default: 'grow' }}
                    className="pf-u-mt-lg"
                >
                    {deployments.map((deployment: ListDeployment) => {
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
                            onClick={() =>
                                fetchDryRun(dryRunConfig, currentPage + 1, filterText, selected)
                            }
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
        );
    }

    return (
        <>
            <Divider />
            {content}
        </>
    );
}

export default CollectionResults;
