import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';
import Table from 'Components/Table';
import Message from 'Components/Message';

class PoliciesPreview extends Component {
    static propTypes = {
        dryrun: PropTypes.shape({
            alerts: PropTypes.arrayOf(
                PropTypes.shape({
                    deployment: PropTypes.string.isRequired
                })
            ).isRequired,
            excluded: PropTypes.arrayOf(
                PropTypes.shape({
                    deployment: PropTypes.string.isRequired
                })
            ).isRequired
        }).isRequired
    };

    renderWarnMessage = () => {
        const message =
            'The policy settings you have selected will generate the following alerts on your system, Please verify that this seems accurate before saving.';
        return <Message message={message} type="warn" />;
    };

    renderPanel = (title, direction) => {
        const icons = {
            up: <Icon.ChevronUp className="h-4 w-4" />,
            down: <Icon.ChevronDown className="h-4 w-4" />
        };

        return (
            <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide cursor-pointer flex justify-between">
                <div>{title}</div>
                <div>{icons[direction]}</div>
            </div>
        );
    };

    renderAlertPreview = () => {
        if (!this.props.dryrun.alerts) return '';
        const title = 'Alert Preview';
        const columns = [
            {
                key: 'deployment',
                keyValueFunc: deployment => deployment,
                label: 'Deployment'
            },
            {
                key: 'violations',
                label: 'Violations',
                keyValueFunc: violations => violations
            }
        ];
        const rows = this.props.dryrun.alerts;

        return (
            <div className="px-3 pb-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderPanel(title, 'up')}
                        triggerWhenOpen={this.renderPanel(title, 'down')}
                        transitionTime={200}
                    >
                        <Table columns={columns} rows={rows} />
                    </Collapsible>
                </div>
            </div>
        );
    };

    renderWhiteListExclusions = () => {
        if (!this.props.dryrun.excluded) return '';
        const title = 'Whitelist Exclusions';
        const columns = [
            {
                key: 'deployment',
                keyValueFunc: deployment => deployment,
                label: 'Deployment'
            }
        ];
        const rows = this.props.dryrun.excluded;

        return (
            <div className="px-3 pb-4">
                <div className="whitelist-exclusions bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderPanel(title, 'up')}
                        triggerWhenOpen={this.renderPanel(title, 'down')}
                        transitionTime={200}
                    >
                        <Table columns={columns} rows={rows} />
                    </Collapsible>
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="bg-base-100">
                {this.renderWarnMessage()}
                {this.renderAlertPreview()}
                {this.renderWhiteListExclusions()}
            </div>
        );
    }
}

export default PoliciesPreview;
