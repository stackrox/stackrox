import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

import { severityLabels } from 'messages/common';

class SeverityTile extends Component {
    static propTypes = {
        severity: PropTypes.string.isRequired,
        count: PropTypes.string.isRequired,
        index: PropTypes.number.isRequired,
        color: PropTypes.string.isRequired
    };

    renderTileContent() {
        const backgroundStyle = {
            backgroundColor: this.props.color
        };
        return (
            <div>
                <div className="absolute pin-l pin-t m-2">
                    <div className="h-3 w-3" style={backgroundStyle} />
                </div>
                <div className="text-4xl text-base font-sans text-primary-500">
                    {this.props.count}
                </div>
                <div className="text-lg text-base font-sans text-primary-500">
                    {severityLabels[this.props.severity]}
                </div>
            </div>
        );
    }

    render() {
        if (this.props.count === 0) {
            return (
                <div
                    className={`flex flex-1 flex-col bg-white border border-base-300 p-4 text-center relative ${
                        this.props.index !== 0 ? 'ml-4' : ''
                    }`}
                >
                    {this.renderTileContent()}
                </div>
            );
        }
        return (
            <Link
                className={`flex flex-1 flex-col bg-white border border-base-300 p-4 text-center relative cursor-pointer no-underline hover:border-base-500 hover:shadow hover:bg-base-100 ${
                    this.props.index !== 0 ? 'ml-4' : ''
                }`}
                to={`/main/violations?severity=${this.props.severity}`}
            >
                {this.renderTileContent()}
            </Link>
        );
    }
}

export default SeverityTile;
