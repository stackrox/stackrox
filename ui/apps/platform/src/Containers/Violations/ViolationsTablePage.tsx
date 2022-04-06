import React, { useEffect, useState, ReactElement } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import Raven from 'raven-js';
import { PageSection, Bullseye, Alert, Divider, Title } from '@patternfly/react-core';

import { actions as alertActions } from 'reducers/alerts';
import { SearchState } from 'reducers/pageSearch';
import { selectors } from 'reducers';
import { fetchAlerts, fetchAlertCount } from 'services/AlertsService';

import useEntitiesByIdsCache from 'hooks/useEntitiesByIdsCache';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import VIOLATION_STATES from 'constants/violationStates';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import { SearchEntry } from 'types/search';

import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';
import useTableSort from 'hooks/useTableSort';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import ViolationsTablePanel from './ViolationsTablePanel';
import tableColumnDescriptor from './violationTableColumnDescriptors';

import './ViolationsTablePage.css';

function runAfter5Seconds(fn: () => void) {
    return new Promise(() => {
        setTimeout(fn, 5000);
    });
}

const violationsPageState = createStructuredSelector<
    SearchState,
    { searchOptions: SearchEntry[]; searchModifiers: SearchEntry[] }
>({
    searchOptions: selectors.getAlertsSearchOptions,
    searchModifiers: selectors.getAlertsSearchModifiers,
});

function ViolationsTablePage(): ReactElement {
    const dispatch = useDispatch();

    const { searchOptions, searchModifiers } = useSelector(violationsPageState);

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(1);
    const [perPage, setPerPage] = useState(50);

    // Handle changes in the currently displayed violations.
    const [currentPageAlerts, setCurrentPageAlerts] = useEntitiesByIdsCache();
    const [currentPageAlertsErrorMessage, setCurrentPageAlertsErrorMessage] = useState('');
    const [alertCount, setAlertCount] = useState(0);

    // To handle page/count refreshing.
    const [pollEpoch, setPollEpoch] = useState(0);
    const [isFetching, setIsFetching] = useState(false);

    // To handle sort options.
    const columns = tableColumnDescriptor;
    const defaultSort = {
        field: 'Violation Time',
        reversed: true,
    };
    const {
        activeSortIndex,
        setActiveSortIndex,
        activeSortDirection,
        setActiveSortDirection,
        sortOption,
    } = useTableSort(columns, defaultSort);

    // Update the isViewFiltered and the value of the selectedAlertId based on changes in search options.
    const hasExecutableFilter =
        searchOptions.length && !searchOptions[searchOptions.length - 1].type;
    const hasNoFilter = !searchOptions.length;

    if (hasExecutableFilter && !isViewFiltered) {
        setIsViewFiltered(true);
        setCurrentPage(1);
    } else if (hasNoFilter && isViewFiltered) {
        setIsViewFiltered(false);
        setCurrentPage(1);
    }

    // When any of the deps to this effect change, we want to reload the alerts and count.
    useEffect(() => {
        if (
            !isFetching &&
            (!searchOptions.length || !searchOptions[searchOptions.length - 1].type)
        ) {
            // Get the alerts that match the search request for the current page.
            setCurrentPageAlertsErrorMessage('');
            setIsFetching(true);
            fetchAlerts(searchOptions, sortOption, currentPage - 1, perPage)
                .then((alerts) => {
                    setCurrentPageAlerts(alerts);
                })
                .catch((error) => {
                    setCurrentPageAlerts([]);
                    const parsedMessage = checkForPermissionErrorMessage(error);
                    setCurrentPageAlertsErrorMessage(parsedMessage);
                })
                .finally(() => {
                    setIsFetching(false);
                });
            // Get the total count of alerts that match the search request.
            fetchAlertCount(searchOptions)
                .then(setAlertCount)
                .catch((error) => {
                    setCurrentPageAlerts([]);
                    const parsedMessage = checkForPermissionErrorMessage(error);
                    setCurrentPageAlertsErrorMessage(parsedMessage);
                });
        }

        // We will update the poll epoch after 5 seconds to force a refresh.
        runAfter5Seconds(() => {
            setPollEpoch(pollEpoch + 1);
        }).catch((error) => Raven.captureException(error));
    }, [
        searchOptions,
        currentPage,
        sortOption,
        pollEpoch,
        setCurrentPageAlerts,
        setCurrentPageAlertsErrorMessage,
        setAlertCount,
        perPage,
    ]);

    // We need to be able to identify which alerts are runtime or attempted, and which are not by id.
    const resolvableAlerts: Set<string> = new Set(
        currentPageAlerts
            .filter(
                (alert) =>
                    alert.lifecycleStage === LIFECYCLE_STAGES.RUNTIME ||
                    alert.state === VIOLATION_STATES.ATTEMPTED
            )
            .map((alert) => alert.id as string)
    );

    const excludableAlerts = currentPageAlerts.filter(
        (alert) =>
            alert.enforcementAction !== ENFORCEMENT_ACTIONS.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT
    );

    const defaultOption = searchModifiers.find((x) => x.value === 'Deployment:');

    function setSearchOptions(options) {
        dispatch(alertActions.setAlertsSearchOptions(options));
    }

    function setSearchSuggestions(suggestions) {
        dispatch(alertActions.setAlertsSearchSuggestions(suggestions));
    }

    return (
        <>
            <PageSection variant="light" id="violations-table">
                <Title headingLevel="h1">Violations</Title>
                <Divider className="pf-u-py-md" />
                <ReduxSearchInput
                    className="w-full theme-light"
                    searchOptions={searchOptions}
                    searchModifiers={searchModifiers}
                    setSearchOptions={setSearchOptions}
                    setSearchSuggestions={setSearchSuggestions}
                    defaultOption={defaultOption}
                    autoCompleteCategories={['ALERTS']}
                />
            </PageSection>
            <PageSection variant="default">
                {currentPageAlertsErrorMessage ? (
                    <Bullseye>
                        <Alert variant="danger" title={currentPageAlertsErrorMessage} />
                    </Bullseye>
                ) : (
                    <PageSection variant="light">
                        <ViolationsTablePanel
                            violations={currentPageAlerts}
                            violationsCount={alertCount}
                            currentPage={currentPage}
                            setCurrentPage={setCurrentPage}
                            resolvableAlerts={resolvableAlerts}
                            excludableAlerts={excludableAlerts}
                            perPage={perPage}
                            setPerPage={setPerPage}
                            activeSortIndex={activeSortIndex}
                            setActiveSortIndex={setActiveSortIndex}
                            activeSortDirection={activeSortDirection}
                            setActiveSortDirection={setActiveSortDirection}
                            columns={columns}
                        />
                    </PageSection>
                )}
            </PageSection>
        </>
    );
}

export default ViolationsTablePage;
