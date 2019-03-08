import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';

import timeWindows from 'constants/timeWindows';

class TimeWindowSelector extends Component {
    static propTypes = {
        activityTimeWindow: PropTypes.string.isRequired,
        setActivityTimeWindow: PropTypes.func.isRequired
    };

    selectTimeWindow = event => {
        const timeWindow = event.target.value;
        this.props.setActivityTimeWindow(timeWindow);
    };

    render() {
        return (
            <div className="flex relative whitespace-no-wrap border-2 rounded-sm mr-2 ml-2 min-h-10 bg-base-100 border-base-300 hover:border-base-400">
                <div className="absolute pin-y ml-2 flex items-center cursor-pointer z-0 pointer-events-none">
                    <Icon.Clock className="h-4 w-4 text-base-500" />
                </div>
                <select
                    className="pl-8 pr-8 truncate text-lg bg-base-100 py-2 text-sm text-base-600 hover:border-base-300 cursor-pointer"
                    onChange={this.selectTimeWindow}
                    value={this.props.activityTimeWindow}
                >
                    {timeWindows.map(window => (
                        <option key={window} value={window}>
                            {window}
                        </option>
                    ))}
                </select>
                <div className="absolute pl-2 pin-y pin-r flex items-center px-2 cursor-pointer z-0 pointer-events-none">
                    <Icon.ChevronDown className="h-4 w-4" />
                </div>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    activityTimeWindow: selectors.getNetworkActivityTimeWindow
});

const mapDispatchToProps = {
    setActivityTimeWindow: pageActions.setNetworkActivityTimeWindow
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(TimeWindowSelector);
