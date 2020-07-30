import React from 'react';
import PropTypes from 'prop-types';
import { Maximize2 } from 'react-feather';

import TileContent from 'Components/TileContent';

const TimelineOverview = ({ dataTestId, counts, onClick }) => {
    return (
        <button
            type="button"
            className="w-full bg-base-100 border border-base-300 border-primary-300 cursor-pointer flex hover:bg-primary-200 hover:border-primary-300 justify-between leading-normal items-stretch"
            onClick={onClick}
            data-testid={dataTestId}
        >
            {counts.map(({ count, text }, index) => (
                <TileContent
                    key={text}
                    superText={count}
                    text={text}
                    className={`p-2 border-dashed border-r ${
                        index === 0 && 'border-l'
                    } border-primary-300 w-full`}
                    textWrap
                />
            ))}
            <TileContent
                className="p-2 w-full"
                icon={<Maximize2 className="border border-primary-300 h-6 p-1 rounded-full w-6" />}
                text="View Graph"
            />
        </button>
    );
};

TimelineOverview.propTypes = {
    dataTestId: PropTypes.string,
    counts: PropTypes.arrayOf(
        PropTypes.shape({
            count: PropTypes.number.isRequired,
            text: PropTypes.string.isRequiredd,
        })
    ),
    onClick: PropTypes.func.isRequired,
};

TimelineOverview.defaultProps = {
    dataTestId: 'timeline-overview',
    counts: [],
};

export default TimelineOverview;
