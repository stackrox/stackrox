import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Table from 'Components/Table';
import CollapsibleCard from 'Components/CollapsibleCard';
import pluralize from 'pluralize';

class Violations extends Component {
    static propTypes = {
        dryrun: PropTypes.shape({
            alerts: PropTypes.arrayOf(
                PropTypes.shape({
                    deployment: PropTypes.string.isRequired
                })
            ).isRequired
        }).isRequired
    };

    render() {
        if (!this.props.dryrun.alerts) return '';
        const title = 'Violations Preview';
        const columns = [
            {
                accessor: 'deployment',
                Header: 'Deployment'
            },
            {
                accessor: 'violations',
                Header: 'Violations',
                Cell: ({ original }) => {
                    let { violations } = original;
                    const numExtraViolations = violations.length - 5;
                    violations = violations.slice(0, 5);
                    return (
                        <div>
                            {violations.map(violation => (
                                <div>- {violation}</div>
                            ))}
                            {numExtraViolations > 0 && (
                                <div>
                                    + {numExtraViolations} more{' '}
                                    {pluralize('violation', numExtraViolations)}
                                </div>
                            )}
                        </div>
                    );
                }
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
    }
}

export default Violations;
