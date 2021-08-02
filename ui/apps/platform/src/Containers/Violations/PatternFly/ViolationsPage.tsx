import React, { useEffect, useState, ReactElement } from 'react';
import { useLocation, useHistory, useParams } from 'react-router-dom';
// import { connect } from 'react-redux';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import Raven from 'raven-js';

import { actions as alertActions } from 'reducers/alerts';
import { SearchEntry } from 'reducers/pageSearch';
import { selectors } from 'reducers';
import { fetchAlerts, fetchAlertCount } from 'services/AlertsService';

import MessageCentered from 'Components/MessageCentered';
import { PageBody } from 'Components/Panel';
// import SidePanelAdjacentArea from 'Components/SidePanelAdjacentArea';
import useEntitiesByIdsCache from 'hooks/useEntitiesByIdsCache';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import VIOLATION_STATES from 'constants/violationStates';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';

// TODO: Remove custom Page Header and use PF search filters instead
import PageHeader from 'Components/PageHeader';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';
import useTableSort from 'hooks/useTableSort';
import ViolationsTablePanel from './ViolationsTablePanel';
// import ViolationsSidePanel from './SidePanel/ViolationsSidePanel';
import tableColumnDescriptor from './violationTableColumnDescriptors';

function runAfter5Seconds(fn: () => void) {
    return new Promise(() => {
        setTimeout(fn, 5000);
    });
}

const violationsPageState = createStructuredSelector<
    any,
    { searchOptions: SearchEntry[]; searchModifiers: SearchEntry[] }
>({
    searchOptions: selectors.getAlertsSearchOptions,
    searchModifiers: selectors.getAlertsSearchModifiers,
});

function ViolationsPage(): ReactElement {
    const { search } = useLocation();
    const history = useHistory();
    const { alertId } = useParams();
    const dispatch = useDispatch();

    const { searchOptions, searchModifiers } = useSelector(violationsPageState);

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes in the currently selected alert
    const [selectedAlertId, setSelectedAlertId] = useState(alertId as string);

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(0);
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
    } else if (hasNoFilter && isViewFiltered) {
        setIsViewFiltered(false);
    }
    if (
        hasExecutableFilter &&
        selectedAlertId &&
        !currentPageAlerts.find((alert) => alert.id === selectedAlertId)
    ) {
        setSelectedAlertId('');
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
                    setCurrentPageAlertsErrorMessage(
                        error.message || 'An unknown error has occurred.'
                    );
                })
                .finally(() => {
                    setIsFetching(false);
                });
            // Get the total count of alerts that match the search request.
            fetchAlertCount(searchOptions)
                .then(setAlertCount)
                .catch((error) => Raven.captureException(error));
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

    // When the selected image changes, update the URL.
    useEffect(() => {
        const urlSuffix = selectedAlertId ? `/${selectedAlertId}` : '';
        history.push({
            pathname: `/main/violations-pf${urlSuffix}`,
            search,
        });
    }, [selectedAlertId, history, search]);

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

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const defaultOption = searchModifiers.find((x) => x.value === 'Deployment:');

    function setSearchOptions(options) {
        dispatch(alertActions.setAlertsSearchOptions(options));
    }

    function setSearchSuggestions(suggestions) {
        dispatch(alertActions.setAlertsSearchSuggestions(suggestions));
    }

    return (
        <>
            <PageHeader header="Violations" subHeader={subHeader}>
                <ReduxSearchInput
                    className="w-full"
                    searchOptions={searchOptions}
                    searchModifiers={searchModifiers}
                    setSearchOptions={setSearchOptions}
                    setSearchSuggestions={setSearchSuggestions}
                    defaultOption={defaultOption}
                    autoCompleteCategories={['ALERTS']}
                />
            </PageHeader>
            <PageBody>
                {currentPageAlertsErrorMessage ? (
                    <MessageCentered type="error">{currentPageAlertsErrorMessage}</MessageCentered>
                ) : (
                    <>
                        <div className="flex-shrink-1 w-full">
                            <ViolationsTablePanel
                                violations={currentPageAlerts}
                                violationsCount={alertCount}
                                // selectedAlertId={selectedAlertId}
                                setSelectedAlertId={setSelectedAlertId}
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
                        </div>
                        {/* {selectedAlertId && (
                            <SidePanelAdjacentArea width="2/5">
                                <ViolationsSidePanel
                                    selectedAlertId={selectedAlertId}
                                    setSelectedAlertId={setSelectedAlertId}
                                />
                            </SidePanelAdjacentArea>
                        )} */}
                    </>
                )}
            </PageBody>
        </>
    );
}

export default ViolationsPage;
