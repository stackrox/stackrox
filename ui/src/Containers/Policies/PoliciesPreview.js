import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Table from 'Components/Table';
import Message from 'Components/Message';
import CollapsibleCard from 'Components/CollapsibleCard';

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
        }).isRequired,
        policyDisabled: PropTypes.bool.isRequired
    };

    renderWarnMessage = () => {
        let message = '';
        if (this.props.policyDisabled) {
            message =
                'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.';
        } else {
            message =
                'The policy settings you have selected will generate violations for the following deployments on your system, Please verify that this seems accurate before saving.';
        }
        return <Message message={message} type="warn" />;
    };

    renderViolationsPreview = () => {
        if (!this.props.dryrun.alerts) return '';
        const title = 'Violations Preview';
        const columns = [
            {
                accessor: 'deployment',
                Header: 'Deployment'
            },
            {
                accessor: 'violations',
                Header: 'Violations'
            }
        ];
        const rows = this.props.dryrun.alerts;
        return (
            <div className="px-3 pb-4">
                <div className="alert-preview bg-base-100 shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title={title}>
                        {rows.length ? (
                            <Table columns={columns} rows={rows} />
                        ) : (
                            <div className="p-3">
                                No violations will be generated for this policy at this time.
                            </div>
                        )}
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderWhitelistedDeployments = () => {
        if (!this.props.dryrun.excluded) return '';
        const title = 'Whitelisted Deployments';
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
                <div className="whitelist-exclusions bg-base-100 shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title={title}>
                        {rows.length ? (
                            <Table columns={columns} rows={rows} />
                        ) : (
                            <div className="p-3">
                                No deployments will be whitelisted at this time.
                            </div>
                        )}
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="bg-base-200">
                {this.renderWarnMessage()}
                {this.renderViolationsPreview()}
                {this.renderWhitelistedDeployments()}
            </div>
        );
    }
}

export default PoliciesPreview;
