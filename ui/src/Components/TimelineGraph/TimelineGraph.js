import React, { useState } from 'react';
import PropTypes from 'prop-types';

import NameList from 'Components/TimelineGraph/NameList';
import MainView from 'Components/TimelineGraph/MainView';
import Minimap from 'Components/TimelineGraph/Minimap';
import Pagination from 'Components/TimelineGraph/Pagination';

const absoluteMinTimeRange = 0;

const TimelineGraph = ({
    data,
    goToNextView,
    currentPage,
    totalSize,
    pageSize,
    onPageChange,
    absoluteMaxTimeRange
}) => {
    const [minTimeRange, setMinTimeRange] = useState(absoluteMinTimeRange);
    const [maxTimeRange, setMaxTimeRange] = useState(absoluteMaxTimeRange);

    const names = data.map(({ type, id, name, subText, hasChildren }) => ({
        type,
        id,
        name,
        subText,
        hasChildren
    }));
    return (
        <div className="flex flex-1 flex-col h-full" data-testid="timeline-graph">
            <div className="flex h-full w-full">
                <div className="w-1/4 border-r border-base-300">
                    <NameList names={names} onClick={goToNextView} />
                </div>
                <div className="w-3/4">
                    <MainView
                        data={data}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                        numRows={pageSize}
                    />
                </div>
            </div>
            <div className="flex border-t border-base-300">
                <div className="w-1/4 p-3 border-r border-base-300 font-700">
                    <Pagination
                        currentPage={currentPage}
                        totalSize={totalSize}
                        pageSize={pageSize}
                        onChange={onPageChange}
                    />
                </div>
                <div className="w-3/4 font-700">
                    <Minimap
                        minTimeRange={absoluteMinTimeRange}
                        setMinTimeRange={setMinTimeRange}
                        maxTimeRange={absoluteMaxTimeRange}
                        setMaxTimeRange={setMaxTimeRange}
                        data={data}
                        numRows={pageSize}
                    />
                </div>
            </div>
        </div>
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
                    type: PropTypes.string.isRequired
                })
            )
        })
    ),
    goToNextView: PropTypes.func.isRequired,
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onPageChange: PropTypes.func.isRequired,
    absoluteMaxTimeRange: PropTypes.number
};

TimelineGraph.defaultProps = {
    data: [],
    absoluteMaxTimeRange: 3600000 * 24 // default to 24 hours
};

export default TimelineGraph;
