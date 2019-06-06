import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import { filterModes, filterLabels } from '../filterModes';

const baseButtonClassName =
    'flex-no-shrink px-2 py-px border-2 rounded-sm uppercase text-xs font-700';
const buttonClassName = `${baseButtonClassName} border-base-400 hover:bg-primary-200 text-base-600`;
const activeButtonClassName = `${baseButtonClassName} bg-primary-300 border-primary-400 hover:bg-primary-200 text-primary-700 border-l-2 border-r-2`;

const Filters = ({ setFilterMode, offset, filterMode }) => {
    function handleChange(mode) {
        return () => {
            setFilterMode(mode);
        };
    }

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
                        filterMode === filterModes.active ? activeButtonClassName : buttonClassName
                    }
                ${filterMode === filterModes.allowed && 'border-r-0'}`}
                    onClick={handleChange(filterModes.active)}
                >
                    {`${filterLabels[filterModes.active]}`}
                </button>
                <button
                    type="button"
                    value={filterMode}
                    className={`${
                        filterMode === filterModes.allowed
                            ? activeButtonClassName
                            : `${buttonClassName} border-l-0 border-r-0`
                    }`}
                    onClick={handleChange(filterModes.allowed)}
                >
                    {`${filterLabels[filterModes.allowed]}`}
                </button>
                <button
                    type="button"
                    value={filterMode}
                    className={`${
                        filterMode === filterModes.all ? activeButtonClassName : buttonClassName
                    }
                ${filterMode === filterModes.allowed && 'border-l-0'}`}
                    onClick={handleChange(filterModes.all)}
                >
                    {`${filterLabels[filterModes.all]}`}
                </button>
            </div>
        </div>
    );
};

Filters.propTypes = {
    setFilterMode: PropTypes.func.isRequired,
    offset: PropTypes.bool.isRequired,
    filterMode: PropTypes.number.isRequired
};

const mapStateToProps = createStructuredSelector({
    offset: selectors.getNetworkWizardOpen,
    filterMode: selectors.getNetworkGraphFilterMode
});

const mapDispatchToProps = {
    setFilterMode: graphActions.setNetworkGraphFilterMode
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Filters);
