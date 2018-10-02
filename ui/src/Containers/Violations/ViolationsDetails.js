import React, { Component } from 'react';
import PropTypes from 'prop-types';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';

const violationDetailsMap = {
    message: { label: 'Message' },
    link: { label: 'CVE Link' }
};

class ViolationsDetails extends Component {
    static propTypes = {
        violations: PropTypes.arrayOf(PropTypes.shape({ message: PropTypes.string.isRequired }))
    };

    static defaultProps = {
        violations: []
    };

    renderViolations = () => {
        const { violations } = this.props;
        if (!violations.length) return 'None';
        return violations.map(violation => {
            if (!violation.message) return null;
            return (
                <KeyValuePairs
                    data={violation}
                    keyValueMap={violationDetailsMap}
                    key={violation.message}
                />
            );
        });
    };

    render() {
        return (
            <div className="w-full pb-5">
                <div className="px-3 pt-5">
                    <div className="bg-base-100 text-primary-600">
                        <CollapsibleCard title="Violations">
                            <div className="h-full px-3">{this.renderViolations()}</div>
                        </CollapsibleCard>
                    </div>
                </div>
            </div>
        );
    }
}

export default ViolationsDetails;
