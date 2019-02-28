import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import filterModes from '../filterModes';
import wizardStages from '../../Wizard/wizardStages';

const baseButtonClassName =
    'flex-no-shrink px-2 py-px border-2 rounded-sm uppercase text-xs font-700';
const buttonClassName = `${baseButtonClassName} border-base-400 hover:bg-primary-200 text-base-600`;
const activeButtonClassName = `${baseButtonClassName} bg-primary-300 border-primary-400 hover:bg-primary-200 text-primary-700 border-l-2 border-r-2`;

class Filters extends Component {
    static propTypes = {
        setFilterMode: PropTypes.func.isRequired,
        offset: PropTypes.bool.isRequired,
        filterMode: PropTypes.number.isRequired
    };

    handleChange = mode => () => {
        this.props.setFilterMode(mode);
    };

    render() {
        const { offset, filterMode } = this.props;
        return (
            <div
                className={`absolute pin-t pin-l px-2 py-2 ${
                    offset ? 'mt-8' : 'mt-2'
                } ml-2 absolute z-1 bg-primary-100 uppercase flex items-center text-sm border-base-400 border-2`}
            >
                <span className="text-base-500 font-700 mr-2">Connections:</span>
                <div className="flex items-center">
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.active
                                ? activeButtonClassName
                                : buttonClassName
                        }
                    ${filterMode === filterModes.allowed && 'border-r-0'}`}
                        onClick={this.handleChange(filterModes.active)}
                    >
                        Active
                    </button>
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.allowed
                                ? activeButtonClassName
                                : `${buttonClassName} border-l-0 border-r-0`
                        }`}
                        onClick={this.handleChange(filterModes.allowed)}
                    >
                        Allowed
                    </button>
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.all ? activeButtonClassName : buttonClassName
                        }
                    ${filterMode === filterModes.allowed && 'border-l-0'}`}
                        onClick={this.handleChange(filterModes.all)}
                    >
                        All
                    </button>
                </div>
            </div>
        );
    }
}

const getSimulatorOn = createSelector(
    [selectors.getNetworkWizardOpen, selectors.getNetworkWizardStage],
    (wizardOpen, wizardStage) => wizardOpen && wizardStage === wizardStages.simulator
);

const mapStateToProps = createStructuredSelector({
    offset: getSimulatorOn,
    filterMode: selectors.getNetworkGraphFilterMode
});

const mapDispatchToProps = {
    setFilterMode: graphActions.setNetworkGraphFilterMode
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Filters);
