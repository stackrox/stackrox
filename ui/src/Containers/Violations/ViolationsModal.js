import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import ReactTooltip from 'react-tooltip';
import dateFns from 'date-fns';

import { actions as alertActions } from 'reducers/alerts';
import { selectors } from 'reducers';
import * as service from 'services/PoliciesService';
import { severityLabels } from 'messages/common';
import Modal from 'Components/Modal';

class ViolationsModal extends Component {
    static propTypes = {
        // TODO-ivan: alertId and alert duplicate source of truth, what would be a better solution?
        alertId: PropTypes.string.isRequired,
        alert: PropTypes.shape({
            id: PropTypes.string.isRequired
        }),
        fetchAlert: PropTypes.func.isRequired,
        onClose: PropTypes.func.isRequired
    };

    static defaultProps = {
        alert: null
    };

    componentDidMount() {
        this.props.fetchAlert(this.props.alertId);
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.alertId !== this.props.alertId) {
            this.props.fetchAlert(this.props.alertId);
        }
    }

    whitelistDeployment = () => {
        const { alert } = this.props;
        // TODO: show spinner on a button while processing the request
        service
            .whitelistDeployment(alert.policy.id, alert.deployment.name)
            .then(this.props.onClose)
            .catch(error => {
                console.error(error);
            });
    };

    renderHeader = alert => (
        <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
            <span className="flex flex-1">
                {alert.deployment.name} ({alert.deployment.id})
            </span>
            <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.props.onClose} />
        </header>
    );

    renderContainer = container => (
        <div key={container.image.name.sha}>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Registry:</span>
                {` `}
                {container.image.name.registry}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Remote:</span>
                {` `}
                {container.image.name.remote}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">SHA:</span>
                {` `}
                {container.image.name.sha}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Tag:</span>
                {` `}
                {container.image.name.tag}
            </div>
        </div>
    );

    renderViolations = alert => (
        <div className="flex py-2 px-3">
            <div className="flex-row font-bold text-primary-500">Violation:</div>
            <div className="flex-row px-1">
                {alert.violations.map(violation => (
                    <div key={`${violation.message}`} className="flex-col break-words">
                        {violation.message}
                    </div>
                ))}
            </div>
        </div>
    );

    renderAlertStatus = alert => {
        const barClass = `px-4 py-2 ${alert.stale ? 'block' : 'hidden'} bg-success-500 text-white`;
        return <div className={barClass}>Alert is no longer applicable</div>;
    };

    renderWhitelistButton = () => (
        <span key="WhitelistDeployment">
            <button
                className="flex py-1 px-2 rounded-sm text-danger-600 hover:text-white hover:bg-danger-400 uppercase text-center text-sm items-center ml-2 mb-2 bg-white border-2 border-danger-400"
                onClick={this.whitelistDeployment}
                data-tip
                data-for="button-WhitelistDeployment"
            >
                <span className="flex items-center">
                    <Icon.X className="h-4 w-4" />
                </span>
                <span>Whitelist Deployment</span>
            </button>
            <ReactTooltip id="button-WhitelistDeployment" type="dark" effect="solid">
                Whitelist deployment for this policy. View whitelists in the policy editing page
            </ReactTooltip>
        </span>
    );

    renderBody = alert => (
        <div className="flex flex-1 overflow-y-scroll">
            <div className="flex flex-col w-full">
                <div className="bg-white m-3 flex-grow">
                    {this.renderViolations(alert)}
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Description:</span>{' '}
                        {alert.policy.description}
                    </div>
                    <div className="flex py-2 px-3">
                        <div className="flex-row font-bold text-primary-500">Rationale:</div>
                        <div className="flex-row px-1">{alert.policy.rationale}</div>
                    </div>
                    <div className="flex py-2 px-3">
                        <div className="flex-row font-bold text-primary-500">Remediation:</div>
                        <div className="flex-row px-1">{alert.policy.remediation}</div>
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Severity:</span>{' '}
                        {severityLabels[alert.policy.severity]}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Date:</span>{' '}
                        {dateFns.format(alert.time, 'MM/DD/YYYY')}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Time:</span>{' '}
                        {dateFns.format(alert.time, 'h:mm:ss A')}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Type:</span>{' '}
                        {alert.deployment.type}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Deployment ID:</span>{' '}
                        {alert.deployment.id}
                    </div>
                    {alert.deployment.containers.map(container => this.renderContainer(container))}
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Categories: </span>
                        {alert.policy.categories.join(', ')}
                    </div>
                    <div>
                        {' '}
                        <hr className="h-px bg-black" />
                    </div>
                    <div className="flex justify-center">{this.renderWhitelistButton()}</div>
                </div>
            </div>
        </div>
    );

    render() {
        const { alert } = this.props;
        if (!alert) return null; // TODO: show loading

        return (
            <Modal isOpen onRequestClose={this.props.onClose} className="w-1/2">
                {this.renderHeader(alert)}
                {this.renderAlertStatus(alert)}
                {this.renderBody(alert)}
            </Modal>
        );
    }
}

const getAlert = createSelector(
    [selectors.getAlertsById, (state, props) => props.alertId],
    (alerts, alertId) => alerts[alertId]
);

const mapStateToProps = createStructuredSelector({
    alert: getAlert
});

const mapDispatchToProps = dispatch => ({
    fetchAlert: alertId => dispatch(alertActions.fetchAlert.request(alertId))
});

export default connect(mapStateToProps, mapDispatchToProps)(ViolationsModal);
