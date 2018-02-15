import React, { Component } from 'react';
import Modal from 'Components/Modal';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import axios from 'axios/index';
import { severityLabels, categoriesLabels } from 'messages/common';
import ReactTooltip from 'react-tooltip';
import dateFns from 'date-fns';

class ViolationsModal extends Component {
    static propTypes = {
        alertId: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            alert: null
        };
    }

    componentDidMount() {
        this.getAlert(this.props.alertId);
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.alertId !== this.props.alertId) {
            this.getAlert(nextProps.alertId);
        }
    }

    getAlert = id => {
        if (!id) return;
        axios
            .get(`/v1/alert/${id}`)
            .then(response => {
                if (!response.data) return;
                this.setState({ alert: response.data });
            })
            .catch(error => {
                console.error(error);
            });
    };

    whitelistDeployment = () => {
        axios.get(`/v1/policies/${this.state.alert.policy.id}`).then(resp => {
            const newPolicy = Object.assign({}, resp.data);
            if (!newPolicy.whitelists) {
                newPolicy.whitelists = [];
            }
            newPolicy.whitelists.push({
                deployment: { name: this.state.alert.deployment.name }
            });
            axios.put(`/v1/policies/${newPolicy.id}`, newPolicy).then(this.props.onClose);
        });
    };

    handleCloseModal = () => {
        this.setState({ alert: null });
        this.props.onClose();
    };

    renderHeader = () => (
        <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
            <span className="flex flex-1">
                {this.state.alert.deployment.name} ({this.state.alert.deployment.id})
            </span>
            <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.handleCloseModal} />
        </header>
    );

    renderContainer = container => (
        <div key={container.image.sha}>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Registry:</span>
                {` `}
                {container.image.registry}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Remote:</span>
                {` `}
                {container.image.remote}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">SHA:</span>
                {` `}
                {container.image.sha}
            </div>
            <div className="py-2 px-3 truncate">
                <span className="font-bold text-primary-500">Tag:</span>
                {` `}
                {container.image.tag}
            </div>
        </div>
    );

    renderViolations = () => (
        <div className="flex py-2 px-3">
            <div className="flex-row font-bold text-primary-500">Violation:</div>
            <div className="flex-row px-1">
                {this.state.alert.violations.map(violation => (
                    <div key={`${violation.message}`} className="flex-col break-words">
                        {violation.message}
                    </div>
                ))}
            </div>
        </div>
    );

    renderAlertStatus = () => {
        const barClass = `px-4 py-2 ${
            this.state.alert.stale ? 'block' : 'hidden'
        } bg-success-500 text-white`;
        return <div className={barClass}>Alert is no longer applicable</div>;
    };

    renderWhiteListButton = () => (
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

    renderBody = () => (
        <div className="flex flex-1 overflow-y-scroll">
            <div className="flex flex-col w-full">
                <div className="bg-white m-3 flex-grow">
                    {this.renderViolations()}
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Description:</span>{' '}
                        {this.state.alert.policy.description}
                    </div>
                    <div className="flex py-2 px-3">
                        <div className="flex-row font-bold text-primary-500">Rationale:</div>
                        <div className="flex-row px-1">{this.state.alert.policy.rationale}</div>
                    </div>
                    <div className="flex py-2 px-3">
                        <div className="flex-row font-bold text-primary-500">Remediation:</div>
                        <div className="flex-row px-1">{this.state.alert.policy.remediation}</div>
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Severity:</span>{' '}
                        {severityLabels[this.state.alert.policy.severity]}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Date:</span>{' '}
                        {dateFns.format(this.state.alert.time, 'MM/DD/YYYY')}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Time:</span>{' '}
                        {dateFns.format(this.state.alert.time, 'h:mm:ss A')}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Type:</span>{' '}
                        {this.state.alert.deployment.type}
                    </div>
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Deployment ID:</span>{' '}
                        {this.state.alert.deployment.id}
                    </div>
                    {this.state.alert.deployment.containers.map(container =>
                        this.renderContainer(container)
                    )}
                    <div className="py-2 px-3 truncate">
                        <span className="font-bold text-primary-500">Categories: </span>
                        {this.state.alert.policy.categories
                            .map(category => categoriesLabels[category])
                            .join(', ')}
                    </div>
                    <div>
                        {' '}
                        <hr className="h-px bg-black" />
                    </div>
                    <div className="flex justify-center">{this.renderWhiteListButton()}</div>
                </div>
            </div>
        </div>
    );

    render() {
        if (!this.state.alert) return '';

        return (
            <Modal isOpen onRequestClose={this.handleCloseModal} className="w-1/2">
                {this.renderHeader()}
                {this.renderAlertStatus()}
                {this.renderBody()}
            </Modal>
        );
    }
}

export default ViolationsModal;
