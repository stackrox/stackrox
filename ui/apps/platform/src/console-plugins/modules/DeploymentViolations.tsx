import { PageSection } from '@patternfly/react-core';
import PluginProvider from 'console-plugins/PluginProvider';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import VIOLATION_STATES from 'constants/violationStates';
import ViolationsTablePanel from 'Containers/Violations/ViolationsTablePanel';
import tableColumnDescriptor from 'Containers/Violations/violationTableColumnDescriptors';
import useEffectAfterFirstRender from 'hooks/useEffectAfterFirstRender';
import useEntitiesByIdsCache from 'hooks/useEntitiesByIdsCache';
import useInterval from 'hooks/useInterval';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import React, { useEffect, useMemo, useState } from 'react';
import { fetchAlerts, fetchAlertCount } from 'services/AlertsService';
import { CancelledPromiseError } from 'services/cancellationUtils';
import { SortOption } from 'types/table';

// Note, react-router information and the current k8s entity among other things are passed
// as props to the component by the console
export default function DeploymentViolations({ obj }) {
    const { name, namespace } = obj.metadata;

    const searchFilter = useMemo(
        () => ({
            Deployment: name,
            Namespace: namespace,
        }),
        [name, namespace]
    );

    // Handle changes in the current table page.
    const { page, perPage, setPage, setPerPage } = useURLPagination(50);

    // Handle changes in the currently displayed violations.
    const [currentPageAlerts, setCurrentPageAlerts] = useEntitiesByIdsCache();
    const [alertCount, setAlertCount] = useState(0);

    // To handle page/count refreshing.
    const [pollEpoch, setPollEpoch] = useState(0);

    // To handle sort options.
    const columns = tableColumnDescriptor;
    const sortFields = useMemo(
        () => columns.flatMap(({ sortField }) => (sortField ? [sortField] : [])),
        [columns]
    );

    const defaultSortOption: SortOption = {
        field: 'Violation Time',
        direction: 'desc',
    };
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    useEffectAfterFirstRender(() => {
        // Prevent viewing a page beyond the maximum page count
        if (page > Math.ceil(alertCount / perPage)) {
            setPage(1);
        }
    }, [alertCount, perPage, setPage]);

    // We will update the poll epoch after 5 seconds to force a refresh of the alert data
    useInterval(() => {
        setPollEpoch(pollEpoch + 1);
    }, 5000);

    // When any of the deps to this effect change, we want to reload the alerts and count.
    useEffect(() => {
        const { request: alertRequest, cancel: cancelAlertRequest } = fetchAlerts(
            searchFilter,
            sortOption,
            page - 1,
            perPage
        );

        // Get the total count of alerts that match the search request.
        const { request: countRequest, cancel: cancelCountRequest } = fetchAlertCount(searchFilter);

        Promise.all([alertRequest, countRequest])
            .then(([alerts, counts]) => {
                setCurrentPageAlerts(alerts);
                setAlertCount(counts);
            })
            .catch((error) => {
                if (error instanceof CancelledPromiseError) {
                    return;
                }
                setCurrentPageAlerts([]);
                setAlertCount(0);
            });

        return () => {
            cancelAlertRequest();
            cancelCountRequest();
        };
    }, [searchFilter, page, sortOption, pollEpoch, setCurrentPageAlerts, setAlertCount, perPage]);

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
    return (
        <PluginProvider>
            <PageSection className="pf-u-m-lg">
                <ViolationsTablePanel
                    violations={currentPageAlerts}
                    violationsCount={alertCount}
                    currentPage={page}
                    setCurrentPage={setPage}
                    resolvableAlerts={resolvableAlerts}
                    excludableAlerts={excludableAlerts}
                    perPage={perPage}
                    setPerPage={setPerPage}
                    getSortParams={getSortParams}
                    columns={columns}
                />
            </PageSection>
        </PluginProvider>
    );
}
