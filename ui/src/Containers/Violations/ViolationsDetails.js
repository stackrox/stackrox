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
            <div className="w-full px-3 py-4">
                <CollapsibleCard title="Violations">
                    <div className="h-full p-3">{this.renderViolations()}</div>
                </CollapsibleCard>
            </div>
        );
    }
}

export default ViolationsDetails;
