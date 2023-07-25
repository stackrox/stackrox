/* eslint-disable react/prop-types */
import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from '@apollo/client';
import { Bullseye } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { useTheme } from 'Containers/ThemeProvider';
import queryService from 'utils/queryService';

import { LIST_PAGE_SIZE, defaultCountKeyMap } from 'constants/workflowPages.constants';
import useCases from 'constants/useCaseTypes';
import vulnMgmtDefaultSorts from '../VulnMgmt.defaultSorts';

export const entityGridContainerBaseClassName =
    'mx-4 grid-dense grid-auto-fit grid grid-gap-4 xl:grid-gap-6 mb-4 xxxl:grid-gap-8';

// to separate out column number related classes from the rest of the grid classes for easy column customization (see policyOverview component)
export const entityGridContainerClassName = `${entityGridContainerBaseClassName} grid-columns-1 md:grid-columns-2 lg:grid-columns-3`;

const useCaseDefaultSorts = {
    [useCases.VULN_MANAGEMENT]: vulnMgmtDefaultSorts,
};

function removeGraphqlAlias(fieldName) {
    const parts = fieldName?.split(':');

    return parts ? parts[0] : '';
}

const WorkflowEntityPage = ({
    ListComponent,
    OverviewComponent,
    entityType,
    entityId,
    entityListType,
    useCase,
    getListQuery,
    overviewQuery,
    queryOptions,
    entityContext,
    search,
    sort,
    page,
    setRefreshTrigger,
}) => {
    const { isDarkMode } = useTheme();

    const enhancedQueryOptions =
        queryOptions && queryOptions.variables ? queryOptions : { variables: {} };
    let query = overviewQuery;
    let fieldName;

    if (entityListType) {
        // sorting stuff
        const appliedSort = sort || useCaseDefaultSorts[useCase][entityListType];
        enhancedQueryOptions.variables.pagination = queryService.getPagination(
            appliedSort,
            page,
            LIST_PAGE_SIZE
        );

        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityType,
            entityListType,
            useCase
        );
        fieldName = listFieldName;
        query = getListQuery(listFieldName, fragmentName, fragment);
    }

    // TODO: if we are ever able to search for k8s and istio vulns, remove this hack
    if (
        enhancedQueryOptions.variables.query &&
        enhancedQueryOptions.variables.query.includes('K8S_CVE')
    ) {
        enhancedQueryOptions.variables.query = enhancedQueryOptions.variables.query.replace(
            /\+?CVE Type:K8S_CVE\+?/,
            ''
        );
    }

    const { loading, data, error } = useQuery(query, enhancedQueryOptions);
    if (loading) {
        return <Loader />;
    }
    if (error) {
        return (
            <Bullseye>
                <EmptyStateTemplate
                    title="Unable to load data"
                    headingLevel="h3"
                    icon={ExclamationTriangleIcon}
                    iconClassName="pf-u-warning-color-100"
                >
                    {error.message}
                </EmptyStateTemplate>
            </Bullseye>
        );
    }
    if (!data || !data.result) {
        return <PageNotFound resourceType={entityType} useCase={useCase} />;
    }
    const { result } = data;

    const listData = entityListType ? result[fieldName] : null;
    const listCountKey = removeGraphqlAlias(defaultCountKeyMap[entityListType]);
    const totalResults = result[listCountKey];
    return entityListType ? (
        <ListComponent
            entityListType={entityListType}
            totalResults={totalResults}
            data={listData}
            search={search}
            sort={sort}
            page={page}
            entityContext={{ ...entityContext, [entityType]: entityId }}
            setRefreshTrigger={setRefreshTrigger}
        />
    ) : (
        <div className={`flex w-full min-h-full ${isDarkMode ? 'bg-base-0' : 'bg-base-200'}`}>
            <div className="w-full min-h-full" id="capture-widgets">
                <OverviewComponent
                    data={result}
                    entityContext={entityContext}
                    setRefreshTrigger={setRefreshTrigger}
                />
            </div>
        </div>
    );
};

WorkflowEntityPage.propTypes = {
    ListComponent: PropTypes.func.isRequired,
    OverviewComponent: PropTypes.func.isRequired,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    useCase: PropTypes.string.isRequired,
    getListQuery: PropTypes.func.isRequired,
    overviewQuery: PropTypes.shape({}).isRequired,
    queryOptions: PropTypes.shape({}),
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    sort: PropTypes.arrayOf(PropTypes.shape({})),
    page: PropTypes.number,
    setRefreshTrigger: PropTypes.func,
};

WorkflowEntityPage.defaultProps = {
    entityListType: null,
    queryOptions: null,
    entityContext: {},
    search: null,
    sort: null,
    page: 1,
    setRefreshTrigger: null,
};

export default WorkflowEntityPage;
