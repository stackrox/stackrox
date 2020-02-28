import React from 'react';
import PropTypes from 'prop-types';

const TimelineGraph = ({ data }) => {
    return (
        <div className="flex flex-1 flex-col">
            <div className="flex w-full">
                <div className="w-1/3 border-r border-base-300">
                    <div className="p-3">
                        <div className="font-700">Show Names here:</div>
                        <ul className="mt-3">
                            {data.map(({ name }) => {
                                return (
                                    <li className="h-20" key={name}>
                                        {name}
                                    </li>
                                );
                            })}
                        </ul>
                    </div>
                </div>
                <div className="w-2/3">
                    <div className="p-3">
                        <div className="font-700">Show Events here:</div>
                        <ul className="mt-3">
                            {data.map(({ name, events }) => {
                                return (
                                    <li className="h-20" key={name}>
                                        {events.map(({ id, timestamp, type }) => {
                                            const text = `"${type}" Event with ID "${id}" showed up at "${timestamp}"`;
                                            return (
                                                <div className="flex flex-no-wrap" key={id}>
                                                    {text}
                                                </div>
                                            );
                                        })}
                                    </li>
                                );
                            })}
                        </ul>
                    </div>
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
