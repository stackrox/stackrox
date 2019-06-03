import React from 'react';
import PropTypes from 'prop-types';

const Tab = ({ text, index, active, onClick }) => (
    <li
        className={`hover:bg-primary-200 ${active ? 'bg-primary-200' : ''} ${
            index !== 0 ? 'border-l border-base-400' : ''
        }`}
    >
        <button type="button" onClick={onClick}>
            <div className="cursor-pointer capitalize p-3">{text}</div>
        </button>
    </li>
);

Tab.propTypes = {
    text: PropTypes.string.isRequired,
    index: PropTypes.number.isRequired,
    active: PropTypes.bool.isRequired,
    onClick: PropTypes.func.isRequired
};

const GroupedTabs = ({ groups, tabs, activeTab, onClick }) => {
    const onClickHandler = datum => () => {
        onClick(datum);
    };
    const groupMapping = tabs.reduce((acc, curr) => {
        acc[curr.group] = [...(acc[curr.group] || []), curr];
        return acc;
    }, {});
    const result = groups
        .filter(group => groupMapping[group])
        .map(group => {
            const grouppedTabs = groupMapping[group];
            return (
                <ul
                    className="list-reset flex ml-4 border-l border-base-400 border-r relative"
                    key={group}
                >
                    {grouppedTabs.map((datum, i) => (
                        <Tab
                            key={datum.value}
                            index={i}
                            text={datum.text}
                            active={activeTab === datum.value}
                            onClick={onClickHandler(datum)}
                        />
                    ))}
                </ul>
            );
        });
    return (
        <ul className="list-reset flex flex-1 border-b border-base-400 px-4 bg-primary-100 uppercase text-sm">
            {result}
        </ul>
    );
};

GroupedTabs.propTypes = {
    groups: PropTypes.arrayOf(PropTypes.string).isRequired,
    tabs: PropTypes.arrayOf(
        PropTypes.shape({
            group: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired,
            string: PropTypes.string.isRequired
        })
    ).isRequired,
    activeTab: PropTypes.shape().isRequired,
    onClick: PropTypes.func.isRequired
};

export default GroupedTabs;
