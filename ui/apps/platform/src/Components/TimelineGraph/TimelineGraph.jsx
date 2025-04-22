import React, { useState } from 'react';
import PropTypes from 'prop-types';

import ClusteredEventsVisibilityContext from 'Components/TimelineGraph/ClusteredEventsVisibilityContext';
import NameList from 'Components/TimelineGraph/NameList';
import MainView from 'Components/TimelineGraph/MainView';
import Minimap from 'Components/TimelineGraph/Minimap';
import Pagination from 'Components/TimelineGraph/Pagination';

const absoluteMinTimeRange = 0;
const defaultAbsoluteMaxTimeRange = 10;
const MARGIN = 20;

const TimelineGraph = ({
    data,
    goToNextView,
    currentPage,
    totalSize,
    pageSize,
    onPageChange,
    absoluteMaxTimeRange,
    showClusteredEvents,
}) => {
    const adjustedAbsoluteMaxTimeRange =
        absoluteMaxTimeRange === 0 ? defaultAbsoluteMaxTimeRange : absoluteMaxTimeRange; // we don't want to show a range of 0 to 0
    const [minTimeRange, setMinTimeRange] = useState(absoluteMinTimeRange);
    const [maxTimeRange, setMaxTimeRange] = useState(adjustedAbsoluteMaxTimeRange);

    const names = data.map(({ type, id, name, subText, hasChildren, drillDownButtonTooltip }) => ({
        type,
        id,
        name,
        subText,
        hasChildren,
        drillDownButtonTooltip,
    }));

    function onSelectionChange(selection) {
        if (!selection) {
            return;
        }
        setMinTimeRange(selection.start);
        setMaxTimeRange(selection.end);
    }

    return (
        <ClusteredEventsVisibilityContext.Provider value={showClusteredEvents}>
            <div className="flex flex-1 flex-col" data-testid="timeline-graph">
                <div className="flex w-full" id="capture-timeline">
                    <div className="w-1/4 min-w-55 border-r border-base-300">
                        <NameList names={names} onClick={goToNextView} />
                    </div>
                    <div>
                        <MainView
                            data={data}
                            minTimeRange={minTimeRange}
                            maxTimeRange={maxTimeRange}
                            absoluteMinTimeRange={absoluteMinTimeRange}
                            absoluteMaxTimeRange={adjustedAbsoluteMaxTimeRange}
                            numRows={pageSize}
                            margin={MARGIN}
                            onZoomChange={onSelectionChange}
                        />
                    </div>
                </div>
                <div className="flex border-t border-base-300">
                    <div className="w-1/4 min-w-55 p-3 border-r border-base-300 font-700">
                        <Pagination
                            currentPage={currentPage}
                            totalSize={totalSize}
                            pageSize={pageSize}
                            onChange={onPageChange}
                        />
                    </div>
                    <div className="font-700">
                        <Minimap
                            minTimeRange={absoluteMinTimeRange}
                            maxTimeRange={adjustedAbsoluteMaxTimeRange}
                            minBrushTimeRange={minTimeRange}
                            maxBrushTimeRange={maxTimeRange}
                            onBrushSelectionChange={onSelectionChange}
                            data={data}
                            numRows={pageSize}
                        />
                    </div>
                </div>
            </div>
        </ClusteredEventsVisibilityContext.Provider>
    );
};

TimelineGraph.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            type: PropTypes.string.isRequired,
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired,
            subText: PropTypes.string.isRequired,
            events: PropTypes.arrayOf(
                PropTypes.shape({
                    id: PropTypes.string.isRequired,
                    differenceInMilliseconds: PropTypes.number.isRequired,
                    edges: PropTypes.arrayOf(PropTypes.shape({})),
                    type: PropTypes.string.isRequired,
                })
            ),
        })
    ),
    goToNextView: PropTypes.func.isRequired,
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onPageChange: PropTypes.func.isRequired,
    absoluteMaxTimeRange: PropTypes.number,
    showClusteredEvents: PropTypes.bool,
};

TimelineGraph.defaultProps = {
    data: [],
    absoluteMaxTimeRange: defaultAbsoluteMaxTimeRange,
    showClusteredEvents: false,
};

export default TimelineGraph;
