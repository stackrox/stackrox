import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const NumberedList = ({ data }) => {
    const list = data.map(({ text, subText, url, component }, i) => {
        const className = `flex items-center py-2 px-2 ${
            i !== 0 ? 'border-t border-base-300' : ''
        } ${url ? 'hover:bg-base-200' : ''}`;
        let content = (
            <>
                <span className="text-sm text-primary-800 font-600 truncate">
                    {i + 1}. {text}&nbsp;
                    {subText && <span className="text-base-500 text-xs">{subText}</span>}
                </span>
                <div className="flex flex-1 justify-end ml-4 whitespace-no-wrap">{component}</div>
            </>
        );
        if (url) {
            content = (
                <Link className="flex items-center no-underline w-full" to={url}>
                    {content}
                </Link>
            );
        }
        return (
            <li key={text} className={className}>
                {content}
            </li>
        );
    });
    return <ul className="list-reset leading-loose">{list}</ul>;
};

NumberedList.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            text: PropTypes.string.isRequired,
            subText: PropTypes.string,
            components: PropTypes.element,
            url: PropTypes.string
        })
    )
};

NumberedList.defaultProps = {
    data: []
};

export default NumberedList;
