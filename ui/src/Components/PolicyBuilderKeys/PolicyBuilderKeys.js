import React from 'react';
import PropTypes from 'prop-types';

import PolicyBuilderKey from 'Components/PolicyBuilderKey';
import CollapsibleSection from 'Components/CollapsibleSection';

function PolicyBuilderKeys({ keys, className }) {
    const categories = {};
    keys.forEach((key) => {
        if (categories[key.category]) {
            categories[key.category].push(key);
        } else {
            categories[key.category] = [key];
        }
    });

    return (
        <div className={`flex flex-col px-3 pt-3 bg-primary-300 ${className}`}>
            <div className="-ml-6 -mr-3 bg-primary-500 mb-2 p-2 rounded-bl rounded-tl text-base-100">
                Drag out a policy field
            </div>
            {Object.keys(categories).map((categoryName) => (
                <CollapsibleSection
                    title={categoryName}
                    key={categoryName}
                    headerClassName="py-1"
                    titleClassName="w-full"
                >
                    {categories[categoryName].map((key) => {
                        return <PolicyBuilderKey key={key.name} fieldKey={key} />;
                    })}
                </CollapsibleSection>
            ))}
        </div>
    );
}

PolicyBuilderKeys.propTypes = {
    className: PropTypes.string,
    keys: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

PolicyBuilderKeys.defaultProps = {
    className: 'w-1/3',
};

export default PolicyBuilderKeys;
