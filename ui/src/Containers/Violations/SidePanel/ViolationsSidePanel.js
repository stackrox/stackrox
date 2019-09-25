import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { fetchAlert } from 'services/AlertsService';

import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import ViolationTabs from './ViolationTabs';
import ViolationNotFound from './ViolationNotFound';

function loadSelectedAlert(selectedAlertId, setSelectedAlert, setIsFetchingSelectedAlert) {
    if (!selectedAlertId) {
        setSelectedAlert(null);
        return;
    }
    setIsFetchingSelectedAlert(true);
    fetchAlert(selectedAlertId).then(
        alert => {
            setSelectedAlert(alert);
            setIsFetchingSelectedAlert(false);
        },
        () => {
            setSelectedAlert(null);
            setIsFetchingSelectedAlert(false);
        }
    );
}

const ViolationsSidePanel = ({ selectedAlertId, setSelectedAlertId }) => {
    // Store the alert we have fetched and whether or not we are currently fetching an alert.
    const [selectedAlert, setSelectedAlert] = useState(null);
    const [isFetchingSelectedAlert, setIsFetchingSelectedAlert] = useState(true);

    // Make updates to the fetching state, and selected alert.
    useEffect(
        () => {
            loadSelectedAlert(selectedAlertId, setSelectedAlert, setIsFetchingSelectedAlert);
        },
        [selectedAlertId, setSelectedAlert, setIsFetchingSelectedAlert]
    );

    // If no alert is selected, nothing to render.
    if (!selectedAlertId) {
        return null;
    }

    // We want to clear the selected alert id on close.
    function unselectAlert() {
        setSelectedAlertId(null);
    }

    // Skip rendering if no alert is there to render.
    let content;
    if (!selectedAlert && !isFetchingSelectedAlert) {
        content = <ViolationNotFound />;
    } else if (!selectedAlert || selectedAlert.id !== selectedAlertId) {
        content = <Loader />;
    } else {
        content = <ViolationTabs alert={selectedAlert} />;
    }

    const header =
        selectedAlert && selectedAlert.deployment
            ? `${selectedAlert.deployment.name} (${selectedAlert.deployment.id})`
            : 'Unknown violation';
    return (
        <Panel
            header={header}
            className="bg-primary-200 z-1 w-full h-full absolute pin-r pin-t min-w-72 md:w-1/2 md:relative"
            onClose={unselectAlert}
        >
            {content}
        </Panel>
    );
};

ViolationsSidePanel.propTypes = {
    selectedAlertId: PropTypes.string,
    setSelectedAlertId: PropTypes.func.isRequired
};

ViolationsSidePanel.defaultProps = {
    selectedAlertId: undefined
};

export default ViolationsSidePanel;
