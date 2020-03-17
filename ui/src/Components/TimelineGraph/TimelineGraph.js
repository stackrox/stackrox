import React, { useState } from 'react';
import PropTypes from 'prop-types';

import NameList from 'Components/TimelineGraph/NameList';
import MainView from 'Components/TimelineGraph/MainView';
import Minimap from 'Components/TimelineGraph/Minimap';

const ABSOLUTE_MIN_TIME_RANGE = 0;
const ABSOLUTE_MAX_TIME_RANGE = 24;

const NUM_ROWS = 10;

const TimelineGraph = ({ data, goToNextView }) => {
    const [minTimeRange, setMinTimeRange] = useState(ABSOLUTE_MIN_TIME_RANGE);
    const [maxTimeRange, setMaxTimeRange] = useState(ABSOLUTE_MAX_TIME_RANGE);

    const names = data.map(({ type, id, name, subText, hasChildren }) => ({
        type,
        id,
        name,
        subText,
        hasChildren
    }));
    return (
        <div className="flex flex-1 flex-col" data-testid="timeline-graph">
            <div className="flex w-full">
                <div className="w-1/4 border-r border-base-300">
                    <NameList names={names} onClick={goToNextView} />
                </div>
                <div className="w-3/4">
                    <MainView
                        data={data}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                        numRows={NUM_ROWS}
                    />
                </div>
            </div>
            <div className="flex border-t border-base-300">
                <div className="w-1/4 p-3 border-r border-base-300 font-700">
                    Show Pagination Controls here...
                </div>
                <div className="w-3/4 font-700">
                    <Minimap
                        minTimeRange={ABSOLUTE_MIN_TIME_RANGE}
                        setMinTimeRange={setMinTimeRange}
                        maxTimeRange={ABSOLUTE_MAX_TIME_RANGE}
                        setMaxTimeRange={setMaxTimeRange}
                        data={data}
                        numRows={NUM_ROWS}
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
                    differenceInHours: PropTypes.number.isRequired,
                    edges: PropTypes.arrayOf(PropTypes.shape({})),
                    type: PropTypes.string.isRequired
                })
            )
        })
    ),
    goToNextView: PropTypes.func.isRequired
};

TimelineGraph.defaultProps = {
    data: []
};

export default TimelineGraph;
