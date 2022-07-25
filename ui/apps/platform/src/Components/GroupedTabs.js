import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { useTheme } from 'Containers/ThemeProvider';

const Tab = ({ text, index, active, to }) => (
    <li
        className={`hover:bg-primary-200 flex flex-grow items-center ${
            active ? 'bg-base-100 text-primary-700' : ''
        } ${index !== 0 ? 'border-l border-base-400' : ''}`}
    >
        <Link to={to} data-testid="tab" className={`w-full no-underline ${active && 'active'}`}>
            <div
                className={`${
                    active ? 'text-primary-700' : 'text-base-500 hover:text-base-600'
                } cursor-pointer uppercase tracking-wide font-700 p-3 flex justify-center`}
            >
                {text}
            </div>
        </Link>
    </li>
);

Tab.propTypes = {
    text: PropTypes.string.isRequired,
    index: PropTypes.number.isRequired,
    active: PropTypes.bool.isRequired,
    to: PropTypes.string.isRequired,
};

const GroupedTabs = ({ groups, tabs, activeTab }) => {
    const { isDarkMode } = useTheme();
    const groupMapping = tabs.reduce((acc, curr) => {
        acc[curr.group] = [...(acc[curr.group] || []), curr];
        return acc;
    }, {});
    const result = groups
        .filter((group) => groupMapping[group])
        .map((group, idx) => {
            const grouppedTabs = groupMapping[group];
            // not showing groups when it's the first (overview) or when there is only one tab child
            const showGroupTab = idx !== 0 && grouppedTabs.length !== 1;
            return (
                <li
                    data-testid="grouped-tab"
                    className={`
                        ${!isDarkMode ? 'bg-primary-100' : 'bg-base-0'} ${
                        idx !== 0 ? 'ml-4' : ''
                    } flex flex-col relative justify-end`}
                    key={group}
                >
                    {showGroupTab && (
                        <span
                            className="truncate absolute top-0 z-10 border-l border-t border-r border-base-400 text-2xs tracking-wide py-1 px-2 rounded-t-lg text-base-500 w-full"
                            style={{ transform: 'translateY(-100%)' }}
                        >
                            {group}
                        </span>
                    )}
                    <ul
                        className={`${
                            showGroupTab ? `flex-1` : ''
                        } flex  border-l border-base-400 border-r h-full`}
                    >
                        {grouppedTabs.map((datum, i) => (
                            <Tab
                                key={datum.value}
                                index={i}
                                text={datum.text}
                                active={activeTab === datum.value}
                                to={datum.to}
                            />
                        ))}
                    </ul>
                </li>
            );
        });
    return (
        <div className="relative">
            <ul
                data-testid="grouped-tabs"
                className={` flex border-b border-base-400 px-4 uppercase text-sm ignore-react-onclickoutside ${
                    !isDarkMode ? 'bg-primary-100' : 'bg-base-0'
                }`}
            >
                {result}
            </ul>
        </div>
    );
};

GroupedTabs.propTypes = {
    groups: PropTypes.arrayOf(PropTypes.string).isRequired,
    tabs: PropTypes.arrayOf(
        PropTypes.shape({
            group: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired,
            text: PropTypes.string.isRequired,
            to: PropTypes.string.isRequired,
        })
    ).isRequired,
    activeTab: PropTypes.string.isRequired,
};

export default GroupedTabs;
