import React, { useEffect, useState } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';

import dialogues from './dialogues';

import ViolationsPageHeader from './ViolationsPageHeader';
import ViolationsTablePanel from './ViolationsTablePanel';
import ViolationsSidePanel from './SidePanel/ViolationsSidePanel';
import ResolveConfirmation from './Dialogues/ResolveConfirmation';
import WhitelistConfirmation from './Dialogues/WhitelistConfirmation';
import TagConfirmation from './Dialogues/TagConfirmation';

function ViolationsPage({
    history,
    location: { search },
    match: {
        params: { alertId }
    }
}) {
    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes in the currently selected alert, and checked alerts.
    const [selectedAlertId, setSelectedAlertId] = useState(alertId);
    const [checkedAlertIds, setCheckedAlertIds] = useState([]);

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(0);
    const [sortOption, setSortOption] = useState({ field: 'Violation Time', reversed: true });

    // Handle changes in the currently displayed violations.
    const [currentAlerts, setCurrentAlerts] = useState([]);
    const [alertCount, setAlertCount] = useState(0);

    // Handle confirmation dialogue being open.
    const [dialogue, setDialogue] = useState(null);

    // When the selected image changes, update the URL.
    useEffect(
        () => {
            const urlSuffix = selectedAlertId ? `/${selectedAlertId}` : '';
            history.push({
                pathname: `/main/violations${urlSuffix}`,
                search
            });
        },
        [selectedAlertId, history, search]
    );

    // We need to be able to identify which alerts are runtime and which are not by id.
    const runtimeAlerts = new Set(
        currentAlerts.filter(alert => alert.lifecycleStage === 'RUNTIME').map(alert => alert.id)
    );

    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <ViolationsPageHeader
                    currentPage={currentPage}
                    sortOption={sortOption}
                    selectedAlertId={selectedAlertId}
                    currentAlerts={currentAlerts}
                    setCurrentAlerts={setCurrentAlerts}
                    setSelectedAlertId={setSelectedAlertId}
                    setAlertCount={setAlertCount}
                    isViewFiltered={isViewFiltered}
                    setIsViewFiltered={setIsViewFiltered}
                />
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 w-full overflow-hidden">
                        <ViolationsTablePanel
                            violations={currentAlerts}
                            violationsCount={alertCount}
                            isViewFiltered={isViewFiltered}
                            setDialogue={setDialogue}
                            selectedAlertId={selectedAlertId}
                            setSelectedAlertId={setSelectedAlertId}
                            checkedAlertIds={checkedAlertIds}
                            setCheckedAlertIds={setCheckedAlertIds}
                            currentPage={currentPage}
                            setCurrentPage={setCurrentPage}
                            setSortOption={setSortOption}
                            runtimeAlerts={runtimeAlerts}
                        />
                    </div>
                    <ViolationsSidePanel
                        selectedAlertId={selectedAlertId}
                        setSelectedAlertId={setSelectedAlertId}
                    />
                </div>
            </div>
            {dialogue === dialogues.whitelist && (
                <WhitelistConfirmation
                    setDialogue={setDialogue}
                    alerts={currentAlerts}
                    checkedAlertIds={checkedAlertIds}
                    setCheckedAlertIds={setCheckedAlertIds}
                />
            )}
            {dialogue === dialogues.resolve && (
                <ResolveConfirmation
                    setDialogue={setDialogue}
                    checkedAlertIds={checkedAlertIds}
                    setCheckedAlertIds={setCheckedAlertIds}
                    runtimeAlerts={runtimeAlerts}
                />
            )}
            {dialogue === dialogues.tag && (
                <TagConfirmation
                    setDialogue={setDialogue}
                    checkedAlertIds={checkedAlertIds}
                    setCheckedAlertIds={setCheckedAlertIds}
                />
            )}
        </section>
    );
}

ViolationsPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired
};

export default ViolationsPage;
