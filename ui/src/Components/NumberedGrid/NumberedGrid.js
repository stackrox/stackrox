import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const NumberedGrid = ({ data }) => {
    const list = data.map(({ text, subText, url, component }, index) => {
        const className = `inline-block w-full py-2 px-2 border-b  border-r border-base-300 ${
            url ? 'hover:bg-base-200' : ''
        }`;
        let content = (
            <div className="flex flex-1 items-center">
                <span className="text-base-600 self-center text-2xl tracking-widest pl-2 pr-4 font-600">
                    {index + 1}
                </span>
                <div className="flex flex-1 flex-col">
                    {subText && (
                        <div className="text-base-500 italic font-600 text-sm mb-1 whitespace-no-wrap truncate">
                            {subText}
                        </div>
                    )}
                    <div className="text-base-600 font-600 text-base mr-4 whitespace-no-wrap truncate">
                        {text}
                    </div>
                    {component && <div className="mt-2">{component}</div>}
                </div>
            </div>
        );
        if (url) {
            content = (
                <Link
                    key={text}
                    to={url}
                    className="flex items-center no-underline text-base-600 hover:bg-base-200 inline-block w-full"
                >
                    {content}
                </Link>
            );
        }
        return <li className={className}>{content}</li>;
    });
    return (
        <ul
            className="list-reset w-full columns-2 columns-gap-0"
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
            subText: PropTypes.string,
            components: PropTypes.element,
            url: PropTypes.string
        })
    )
};

NumberedGrid.defaultProps = {
    data: []
};

export default NumberedGrid;
