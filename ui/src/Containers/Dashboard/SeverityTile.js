import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

const severityLabels = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low'
};

class SeverityTile extends Component {
    static propTypes = {
        severity: PropTypes.string.isRequired,
        count: PropTypes.number.isRequired,
        color: PropTypes.string.isRequired,
        history: PropTypes.shape({
            push: PropTypes.func.isRequired
        }).isRequired,
        index: PropTypes.number.isRequired
    };

    goToViolationsPage = () => {
        this.props.history.push(`/violations?severity=${this.props.severity}`);
    }

    render() {
        const backgroundStyle = {
            backgroundColor: this.props.color
        };
        return (
            <Link
                className={`flex flex-1 flex-col bg-white border border-base-300 p-4 text-center relative cursor-pointer no-underline hover:border-base-500 hover:shadow hover:bg-base-100 ${this.props.index !== 0 ? 'ml-4' : ''}`}
                to={`/violations?severity=${this.props.severity}`}
            >
                <div className="absolute pin-l pin-t m-2">
                    <div className="h-3 w-3" style={backgroundStyle} />
                </div>
                <div className="text-4xl text-base font-sans text-primary-500">{this.props.count}</div>
                <div className="text-lg text-base font-sans text-primary-500">{severityLabels[this.props.severity]}</div>
            </Link>
        );
    }
}

export default SeverityTile;
