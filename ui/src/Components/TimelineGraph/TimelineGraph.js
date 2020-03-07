import React from 'react';
import PropTypes from 'prop-types';

import NameList from 'Components/TimelineGraph/NameList';
import EventsGraph from 'Components/TimelineGraph/EventsGraph';

const TimelineGraph = ({ data, goToNextView }) => {
    const names = data.map(({ type, id, name, subText, hasChildren }) => ({
        type,
        id,
        name,
        subText,
        hasChildren
    }));
    return (
        <div className="flex flex-1 flex-col">
            <div className="flex w-full">
                <div className="w-1/4 border-r border-base-300">
                    <NameList names={names} onClick={goToNextView} />
                </div>
                <div className="w-3/4">
                    <EventsGraph data={data} />
                </div>
            </div>
            <div className="flex border-t border-base-300">
                <div className="w-1/4 p-3 border-r border-base-300 font-700">
                    Show Pagination Controls here...
                </div>
                <div className="w-3/4 p-3 font-700">Show Minimap here...</div>
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
