import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Table from 'Components/Table';
import CollapsibleCard from 'Components/CollapsibleCard';

class ExcludedScopes extends Component {
    static propTypes = {
        dryrun: PropTypes.shape({
            excluded: PropTypes.arrayOf(
                PropTypes.shape({
                    deployment: PropTypes.string.isRequired,
                })
            ).isRequired,
        }).isRequired,
    };

    render() {
        if (!this.props.dryrun || !this.props.dryrun.excluded) {
            return '';
        }

        const title = 'Excluded Deployments';
        const columns = [
            {
                key: 'deployment',
                keyValueFunc: (deployment) => deployment,
                label: 'Deployment',
            },
        ];
        const rows = this.props.dryrun.excluded;

        return (
            <div className="px-3 pb-4">
                <div className="whitelist-exclusions bg-base-100 shadow text-primary-600 tracking-wide">
                    <CollapsibleCard title={title}>
                        {rows.length ? (
                            <Table columns={columns} rows={rows} />
                        ) : (
                            <div className="p-3">No deployments will be excluded at this time.</div>
                        )}
                    </CollapsibleCard>
                </div>
            </div>
        );
    }
}

export default ExcludedScopes;
