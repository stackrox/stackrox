import React, { useMemo, useCallback } from 'react';
import { useLocation } from 'react-router-dom';
import {
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { PolicySeverity } from 'types/policy.proto';

import useWidgetConfig from 'hooks/useWidgetConfig';
import useAlertGroups from '../hooks/useAlertGroups';
import WidgetCard from './WidgetCard';
import NoDataEmptyState from './NoDataEmptyState';
import ViolationsByPolicyCategoryChart, { Config } from './ViolationsByPolicyCategoryChart';
import OptionsDropdown from './OptionsDropdown';

const fieldIdPrefix = 'policy-category-violations';

const defaultHiddenSeverities = ['MEDIUM_SEVERITY', 'LOW_SEVERITY'] as const;

const defaultConfig = {
    sortType: 'Severity',
    lifecycle: 'ALL',
    hiddenSeverities: defaultHiddenSeverities,
} as const;

function ViolationsByPolicyCategory() {
    const { pathname } = useLocation();
    const { searchFilter } = useURLSearch();

    const [{ sortType, lifecycle, hiddenSeverities }, updateConfig] = useWidgetConfig<Config>(
        'ViolationsByPolicyCategory',
        pathname,
        defaultConfig
    );

    const hiddenSeveritySet = useMemo(() => new Set(hiddenSeverities), [hiddenSeverities]);

    const onHiddenSeverityUpdate = useCallback(
        (newHidden: Set<PolicySeverity>) =>
            updateConfig({ hiddenSeverities: Array.from(newHidden) }),
        [updateConfig]
    );

    const queryFilter = { ...searchFilter };
    if (lifecycle === 'DEPLOY') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.DEPLOY;
    } else if (lifecycle === 'RUNTIME') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.RUNTIME;
    }
    const query = getRequestQueryStringForSearchFilter(queryFilter);
    const { data: alertGroups, loading, error } = useAlertGroups(query, 'CATEGORY');

    return (
        <WidgetCard
            isLoading={loading}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">Policy violations by category</Title>
                    </FlexItem>
                    <FlexItem>
                        <OptionsDropdown toggleId={`${fieldIdPrefix}-options-toggle`}>
                            <Form className="pf-u-px-md pf-u-py-sm">
                                <FormGroup fieldId={`${fieldIdPrefix}-sort-by`} label="Sort by">
                                    <ToggleGroup aria-label="Sort data by highest severity counts or highest total violations">
                                        <ToggleGroupItem
                                            className="pf-u-font-weight-normal"
                                            text="Severity"
                                            buttonId={`${fieldIdPrefix}-sort-by-severity`}
                                            isSelected={sortType === 'Severity'}
                                            onChange={() => updateConfig({ sortType: 'Severity' })}
                                        />
                                        <ToggleGroupItem
                                            text="Total"
                                            buttonId={`${fieldIdPrefix}-sort-by-total`}
                                            isSelected={sortType === 'Total'}
                                            onChange={() => updateConfig({ sortType: 'Total' })}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                                <FormGroup
                                    fieldId={`${fieldIdPrefix}-lifecycle`}
                                    label="Policy Lifecycle"
                                >
                                    <ToggleGroup aria-label="Filter by policy lifecycle">
                                        <ToggleGroupItem
                                            text="All"
                                            buttonId={`${fieldIdPrefix}-lifecycle-all`}
                                            isSelected={lifecycle === 'ALL'}
                                            onChange={() => updateConfig({ lifecycle: 'ALL' })}
                                        />
                                        <ToggleGroupItem
                                            text="Deploy"
                                            buttonId={`${fieldIdPrefix}-lifecycle-deploy`}
                                            isSelected={lifecycle === 'DEPLOY'}
                                            onChange={() => updateConfig({ lifecycle: 'DEPLOY' })}
                                        />
                                        <ToggleGroupItem
                                            text="Runtime"
                                            buttonId={`${fieldIdPrefix}-lifecycle-runtime`}
                                            isSelected={lifecycle === 'RUNTIME'}
                                            onChange={() => updateConfig({ lifecycle: 'RUNTIME' })}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                            </Form>
                        </OptionsDropdown>
                    </FlexItem>
                </Flex>
            }
        >
            {alertGroups && alertGroups.length > 0 ? (
                <ViolationsByPolicyCategoryChart
                    alertGroups={alertGroups}
                    sortType={sortType}
                    lifecycle={lifecycle}
                    searchFilter={searchFilter}
                    hiddenSeverities={hiddenSeveritySet}
                    setHiddenSeverities={onHiddenSeverityUpdate}
                />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicyCategory;
