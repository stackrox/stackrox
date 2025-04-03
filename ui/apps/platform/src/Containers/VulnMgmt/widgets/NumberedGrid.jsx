import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const NumberedGrid = ({ data }) => {
    const stacked = data.length < 4;
    const list = data.map(({ text, url, component }, index) => {
        const className = `inline-block w-full px-2 border-b border-base-300 ${
            url ? 'hover:bg-base-200 cursor-pointer' : ''
        } ${stacked ? 'py-4' : 'py-2 border-r'}`;
        const content = (
            <div className="flex flex-1 items-center">
                <span className="text-base-600 self-center pl-2 pr-4">{index + 1}</span>
                <div className={`flex flex-1 ${stacked ? 'justify-between' : 'flex-col'}`}>
                    <Link to={url} className="flex items-center mr-4 whitespace-nowrap truncate">
                        {text}
                    </Link>
                    {component && <div className={`${stacked ? '' : 'mt-2'}`}>{component}</div>}
                </div>
            </div>
        );

        return (
            <li key={text} className={className}>
                {content}
            </li>
        );
    });
    return (
        <ul
            className={`w-full ${stacked ? 'columns-1' : 'columns-2'} columns-gap-0`}
            style={{ columnRule: '1px solid var(--base-300)' }}
        >
            {list}
        </ul>
    );
};

NumberedGrid.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            text: PropTypes.string.isRequired,
            components: PropTypes.element.isRequired,
            url: PropTypes.string.isRequired,
        })
    ),
};

NumberedGrid.defaultProps = {
    data: [],
};

export default NumberedGrid;
