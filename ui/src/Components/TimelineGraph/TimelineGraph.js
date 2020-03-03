import React from 'react';
import PropTypes from 'prop-types';

import NameList from 'Components/TimelineGraph/NameList';

const TimelineGraph = ({ data }) => {
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
                <div className="w-1/3 border-r border-base-300">
                    <NameList names={names} />
                </div>
                <div className="w-2/3">
                    <ul className="">
                        {data.map(({ name, events }, index) => {
                            return (
                                <li
                                    className={`flex h-12 items-center justify-center px-4 ${
                                        index !== 0 ? 'border-t border-base-300' : ''
                                    }`}
                                    key={name}
                                >
                                    <div className="flex flex-no-wrap">
                                        {events
                                            .map(({ id, type }) => `"${type}" Event "${id}""`)
                                            .join(', ')}
                                    </div>
                                </li>
                            );
                        })}
                    </ul>
                </div>
            </div>
            <div className="flex border-t border-base-300">
                <div className="w-1/3 p-3 border-r border-base-300 font-700">
                    Show Pagination Controls here...
                </div>
                <div className="w-2/3 p-3 font-700">Show Minimap here...</div>
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
                    timestamp: PropTypes.string.isRequired,
                    edges: PropTypes.arrayOf(PropTypes.shape({})),
                    type: PropTypes.string.isRequired
                })
            )
        })
    )
};

TimelineGraph.defaultProps = {
    data: []
};

export default TimelineGraph;
