import React from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

const NumberedGrid = ({ data, history }) => {
    const onClick = (url) => () => {
        if (!url) {
            return null;
        }
        history.push(url);
        return 0;
    };

    const stacked = data.length < 4;
    const list = data.map(({ text, subText, url, component }, index) => {
        const className = `inline-block w-full px-2 border-b border-base-300 ${
            url ? 'hover:bg-base-200 cursor-pointer' : ''
        } ${stacked ? 'py-4' : 'py-2 border-r'}`;
        const content = (
            <div className="flex flex-1 items-center">
                <span className="text-base-600 self-center pl-2 pr-4">{index + 1}</span>
                <div className={`flex flex-1 ${stacked ? 'justify-between' : 'flex-col'}`}>
                    {subText && (
                        <div className="text-base-500 text-sm mb-1 whitespace-nowrap truncate">
                            {subText}
                        </div>
                    )}
                    <div className="text-base-600 flex items-center text-base mr-4 whitespace-nowrap truncate">
                        {text}
                    </div>
                    {component && <div className={`${stacked ? '' : 'mt-2'}`}>{component}</div>}
                </div>
            </div>
        );

        return (
            <li key={text} className={className} onClick={onClick(url)}>
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
            subText: PropTypes.string,
            components: PropTypes.element,
            url: PropTypes.string,
        })
    ),
    history: ReactRouterPropTypes.history.isRequired,
};

NumberedGrid.defaultProps = {
    data: [],
};

export default withRouter(NumberedGrid);
