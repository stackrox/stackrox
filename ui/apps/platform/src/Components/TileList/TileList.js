import React from 'react';
import PropTypes from 'prop-types';

import { useTheme } from 'Containers/ThemeProvider';
import TileLink from 'Components/TileLink';

const TileList = ({ items, title }) => {
    const { isDarkMode } = useTheme();
    return (
        <div
            className={`text-base-600 rounded border mx-2 my-3 ${
                !isDarkMode
                    ? 'bg-primary-200 border-primary-400'
                    : 'bg-tertiary-200 border-tertiary-300'
            }`}
        >
            {title !== '' && (
                <h3
                    className={`border-b text-xs text-base-600 uppercase text-center tracking-wide p-1 leading-normal font-700 ${
                        !isDarkMode ? 'border-base-400' : 'border-tertiary-400'
                    }`}
                >
                    {title}
                </h3>
            )}
            <ul className="pb-2">
                {items.map(({ label, count, url, entity }) => (
                    <li className="p-2 pb-0" key={label}>
                        <TileLink
                            colorClasses={` ${
                                !isDarkMode
                                    ? 'border-primary-400 hover:bg-primary-200 rounded'
                                    : 'rounded bg-tertiary-200 border-tertiary-300 hover:bg-tertiary-100 hover:border-tertiary-400'
                            }  `}
                            superText={count}
                            text={label}
                            url={url}
                            dataTestId={`${entity}-tile-link`}
                        />
                    </li>
                ))}
            </ul>
        </div>
    );
};

TileList.propTypes = {
    items: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string,
            count: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
            url: PropTypes.string,
        })
    ).isRequired,
    title: PropTypes.string,
};

TileList.defaultProps = {
    title: '',
};

export default TileList;
