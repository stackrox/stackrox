import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { Link } from 'react-router-dom';

import { selectors } from 'reducers';
import { types as alertTypes } from 'reducers/alerts';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Panel from 'Components/Panel';
import LoadingSection from 'Components/LoadingSection';
import { Details as EnforcementDetails } from 'Containers/Violations/Enforcement/Details';
import Message from 'Components/Message';
import DeploymentDetails from '../Risk/DeploymentDetails';
import ViolationsDetails from './ViolationsDetails';
import { Panel as PolicyDetails } from '../Policies/Wizard/Details/Panel';

const ViolationsPanelContent = ({ alert, isLoading }) => {
    const riskPanelTabs = [
        { text: 'Violation' },
        { text: 'Enforcement' },
        { text: 'Deployment' },
        { text: 'Policy' }
    ];
    const message = (
        <div>
            Violation not found. This violation may have been deleted due to &nbsp;
            <Link to="/main/systemconfig" className="text-primary-700">
                data retention settings
            </Link>
        </div>
    );
    const content = alert ? (
        <Tabs headers={riskPanelTabs}>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <ViolationsDetails
                        violations={alert.violations}
                        processViolation={alert.processViolation}
                    />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <EnforcementDetails listAlert={alert} />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <DeploymentDetails deployment={alert.deployment} />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <PolicyDetails wizardPolicy={alert.policy} />
                </div>
            </TabContent>
        </Tabs>
    ) : (
        <div className="h-full flex-1 bg-base-200 border-r border-l border-b border-base-400 p-3">
            <Message message={message} type="error" />
        </div>
    );

    return isLoading ? <LoadingSection /> : content;
};

ViolationsPanelContent.propTypes = {
    alert: PropTypes.shape({
        id: PropTypes.string
    }),
    isLoading: PropTypes.bool
};

ViolationsPanelContent.defaultProps = {
    alert: null,
    isLoading: false
};

const ViolationsPanel = ({ alertId, alert, onClose, isLoading }) => {
    if (!alertId) return null;

    const header =
        alert && alert.deployment
            ? `${alert.deployment.name} (${alert.deployment.id})`
            : 'Unknown violation';
    return (
        <Panel
            header={header}
            className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t min-w-72 md:w-1/2 md:relative"
            onClose={onClose}
        >
            <ViolationsPanelContent alert={alert} isLoading={isLoading} />
        </Panel>
    );
};

ViolationsPanel.propTypes = {
    alertId: PropTypes.string,
    alert: PropTypes.shape({
        id: PropTypes.string
    }),
    onClose: PropTypes.func.isRequired,
    isLoading: PropTypes.bool
};

ViolationsPanel.defaultProps = {
    alertId: '',
    alert: null,
    isLoading: false
};

const getAlert = createSelector(
    [selectors.getFilteredAlerts, (state, props) => props.alertId],
    (alerts, alertId) => alerts.find(alert => alert.id === alertId)
);

const mapStateToProps = (_outerState, ownProps) => {
    return createStructuredSelector({
        alertId: () => ownProps.alertId,
        alert: getAlert,
        isLoading: state => selectors.getLoadingStatus(state, alertTypes.FETCH_ALERTS)
    });
};

export default connect(mapStateToProps)(ViolationsPanel);
