import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { Activity, Maximize2 } from 'react-feather';

import TileContent from 'Components/TileContent';

const TimelineOverview = ({ type, total, counts, onClick }) => {
    return (
        <button
            type="button"
            className="w-full bg-base-100 border border-base-300 border-primary-300 cursor-pointer flex hover:bg-primary-200 hover:border-primary-300 justify-between leading-normal items-stretch"
            onClick={onClick}
        >
            <TileContent
                className={
                    counts.length === 0 ? 'p-2 border-dashed border-r border-primary-300' : 'p-2'
                }
                icon={<Activity className="border border-primary-300 h-6 p-1 rounded-full w-6" />}
                text={`${total} ${pluralize(type, total)}`}
            />
            {counts.map(({ count, text }, index) => (
                <TileContent
                    key={text}
                    superText={count}
                    text={text}
                    className={`p-2 border-dashed border-r ${index === 0 &&
                        'border-l'} border-primary-300`}
                />
            ))}
            <TileContent
                className="p-2"
                icon={<Maximize2 className="border border-primary-300 h-6 p-1 rounded-full w-6" />}
                text="View Graph"
            />
        </button>
    );
};

TimelineOverview.propTypes = {
    type: PropTypes.string.isRequired,
    total: PropTypes.number.isRequired,
    counts: PropTypes.arrayOf(
        PropTypes.shape({
            count: PropTypes.number.isRequired,
            text: PropTypes.string.isRequiredd
        })
    ),
    onClick: PropTypes.func.isRequired
};

TimelineOverview.defaultProps = {
    counts: []
};

export default TimelineOverview;
