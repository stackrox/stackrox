import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';

const Labels = ({ labels }) => (
    <ul className={` ${labels.length > 4 ? `columns-2` : ``} list-reset p-3 w-full leading-normal`}>
        {labels.map(label => (
            <li
                key={label.key}
                className="border-b border-base-300 p-2 truncate"
                style={{
                    columnBreakInside: 'avoid',
                    pageBreakInside: 'avoid'
                }}
            >
                <Tooltip
                    overlayClassName="w-1/4 pointer-events-none"
                    placement="top"
                    overlay={
                        <div>
                            {' '}
                            {label.key} : {label.value || '""'}
                        </div>
                    }
                    mouseLeaveDelay={0}
                >
                    <h1 className="text-base font-600 word-break truncate">
                        {' '}
                        {label.key} : {label.value || '""'}
                    </h1>
                </Tooltip>
            </li>
        ))}
    </ul>
);

Labels.propTypes = {
    labels: PropTypes.arrayOf({}).isRequired
};

export default Labels;
