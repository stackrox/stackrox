import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import { filterModes, filterLabels } from 'constants/networkFilterModes';
import NamespaceEdgeFilter from './NamespaceEdgeFilter';

const baseButtonClassName =
    'flex-shrink-0 px-2 py-px border-2 rounded-sm uppercase text-xs font-700';
const buttonClassName = `${baseButtonClassName} border-base-400 hover:bg-primary-200 text-base-600`;
const activeButtonClassName = `${baseButtonClassName} bg-primary-300 border-primary-400 hover:bg-primary-200 text-primary-700 border-l-2 border-r-2`;

const Filters = ({
    setFilterMode,
    offset,
    sidePanelOpen,
    filterMode,
    showNamespaceFlows,
    setShowNamespaceFlows,
}) => {
    function handleChange(mode) {
        return () => {
            setFilterMode(mode);
        };
    }

    return (
        <div
            className={`flex absolute top-0 left-0 ${offset ? 'mt-10' : 'mt-2'} ${
                sidePanelOpen ? 'flex-col' : ''
            } ml-2 absolute z-1`}
        >
            <div className="p-2 bg-primary-100 flex items-center text-sm border-base-400 border-2">
                <span className="text-base-500 font-700 mr-2">Flows:</span>
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
                        onClick={handleChange(filterModes.active)}
                        data-testid="network-connections-filter-active"
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
                        data-testid="network-connections-filter-allowed"
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
                        data-testid="network-connections-filter-all"
                    >
                        {`${filterLabels[filterModes.all]}`}
                    </button>
                </div>
            </div>
            <div
                className={`px-2 py-1 bg-primary-100 flex items-center text-sm border-base-400 border-2 ${
                    sidePanelOpen ? 'mt-1' : 'ml-1'
                }`}
            >
                <NamespaceEdgeFilter
                    selectedState={showNamespaceFlows}
                    setFilter={setShowNamespaceFlows}
                />
            </div>
        </div>
    );
};

Filters.propTypes = {
    setFilterMode: PropTypes.func.isRequired,
    offset: PropTypes.bool,
    sidePanelOpen: PropTypes.bool.isRequired,
    filterMode: PropTypes.number.isRequired,
    showNamespaceFlows: PropTypes.string.isRequired,
    setShowNamespaceFlows: PropTypes.string.isRequired,
};

Filters.defaultProps = {
    offset: false,
};

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getNetworkSidePanelOpen,
    filterMode: selectors.getNetworkGraphFilterMode,
});

const mapDispatchToProps = {
    setFilterMode: graphActions.setNetworkGraphFilterMode,
};

export default connect(mapStateToProps, mapDispatchToProps)(Filters);
