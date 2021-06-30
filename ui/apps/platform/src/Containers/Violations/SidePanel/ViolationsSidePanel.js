import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import Loader from 'Components/Loader';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import { fetchAlert } from 'services/AlertsService';

import ViolationTabs from './ViolationTabs';
import ViolationNotFound from './ViolationNotFound';

const ViolationsSidePanel = ({ selectedAlertId, setSelectedAlertId }) => {
    // Store the alert we have fetched and whether or not we are currently fetching an alert.
    const [selectedAlert, setSelectedAlert] = useState(null);
    const [isFetchingSelectedAlert, setIsFetchingSelectedAlert] = useState(false);

    // Make updates to the fetching state, and selected alert.
    useEffect(() => {
        if (!selectedAlertId) {
            setSelectedAlert(null);
            return;
        }
        setIsFetchingSelectedAlert(true);
        fetchAlert(selectedAlertId).then(
            (alert) => {
                setSelectedAlert(alert);
                setIsFetchingSelectedAlert(false);
            },
            () => {
                setSelectedAlert(null);
                setIsFetchingSelectedAlert(false);
            }
        );
    }, [selectedAlertId, setSelectedAlert, setIsFetchingSelectedAlert]);

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
    const { resource, deployment, commonEntityInfo } = selectedAlert || {};
    const { name } = resource || deployment || {};

    const getResourceId = () => {
        const { resourceType } = resource || commonEntityInfo || {};
        if (!resourceType) {
            return `(${selectedAlert.deployment.id})`;
        }
        switch (resourceType) {
            case 'DEPLOYMENT':
                return `(${selectedAlert.deployment.id})`;
            case 'NAMESPACE':
                return `(${selectedAlert.resource.namespaceId})`;
            case 'CLUSTER':
                return `(${selectedAlert.resource.clusterId})`;
            default:
                return '';
        }
    };
    const header = name ? `${name} ${getResourceId()}` : 'Unknown violation';

    /*
     * For border color compatible with background color of SidePanelAdjacentArea:
     */
    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle isUpperCase={false} testid="panel-header" text={header} />
                <PanelHeadEnd>
                    <CloseButton onClose={unselectAlert} className="border-base-400 border-l" />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>{content}</PanelBody>
        </PanelNew>
    );
};

ViolationsSidePanel.propTypes = {
    selectedAlertId: PropTypes.string,
    setSelectedAlertId: PropTypes.func.isRequired,
};

ViolationsSidePanel.defaultProps = {
    selectedAlertId: undefined,
};

export default ViolationsSidePanel;
