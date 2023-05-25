import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import upperFirst from 'lodash/upperFirst';

import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';
import { filterModes, filterLabels } from 'constants/networkFilterModes';
import NamespaceEdgeFilter, { NamespaceEdgeFilterState } from './NamespaceEdgeFilter';

const baseButtonClassName =
    'flex-shrink-0 px-2 py-px border-2 border-base-400 rounded-sm text-base-600';
const activeButtonClassName = `${baseButtonClassName} bg-primary-200 font-700 border-l-2 border-r-2`;

type FiltersProps = {
    setFilterMode: (mode) => void;
    sidePanelOpen: boolean;
    filterMode: number;
    showNamespaceFlows: NamespaceEdgeFilterState;
    setShowNamespaceFlows: (value) => void;
};

function Filters({
    setFilterMode,
    sidePanelOpen,
    filterMode,
    showNamespaceFlows,
    setShowNamespaceFlows,
}: FiltersProps): ReactElement {
    function handleChange(mode) {
        return () => {
            setFilterMode(mode);
        };
    }

    return (
        <div
            className={`flex absolute top-0 left-0 mt-2 ${
                sidePanelOpen ? 'flex-col' : ''
            } ml-2 absolute z-1`}
        >
            <div className="p-2 bg-base-100 text-base-600 flex items-center text-sm border-base-400 border-2">
                <span className="mr-2">Flows:</span>
                <div className="flex items-center">
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.active
                                ? activeButtonClassName
                                : baseButtonClassName
                        }
                ${filterMode === filterModes.allowed ? 'border-r-0' : ''}`}
                        onClick={handleChange(filterModes.active)}
                        data-testid="network-connections-filter-active"
                    >
                        {upperFirst(filterLabels[filterModes.active])}
                    </button>
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.allowed
                                ? activeButtonClassName
                                : `${baseButtonClassName} border-l-0 border-r-0`
                        }`}
                        onClick={handleChange(filterModes.allowed)}
                        data-testid="network-connections-filter-allowed"
                    >
                        {upperFirst(filterLabels[filterModes.allowed])}
                    </button>
                    <button
                        type="button"
                        value={filterMode}
                        className={`${
                            filterMode === filterModes.all
                                ? activeButtonClassName
                                : baseButtonClassName
                        }
                ${filterMode === filterModes.allowed ? 'border-l-0' : ''}`}
                        onClick={handleChange(filterModes.all)}
                        data-testid="network-connections-filter-all"
                    >
                        {upperFirst(filterLabels[filterModes.all])}
                    </button>
                </div>
            </div>
            <div
                className={`px-2 py-1 bg-base-100 flex items-center border-base-400 border-2 ${
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
}

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    filterMode: selectors.getNetworkGraphFilterMode,
});

const mapDispatchToProps = {
    setFilterMode: graphActions.setNetworkGraphFilterMode,
};

export default connect(mapStateToProps, mapDispatchToProps)(Filters);
