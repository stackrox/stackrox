import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';

import PolicyBuilderKey from 'Components/PolicyBuilderKey';
import CollapsibleSection from 'Components/CollapsibleSection';

function getKeysByCategory(keys) {
    const categories = {};
    keys.forEach((key) => {
        const { category } = key;
        if (categories[category]) {
            categories[category].push(key);
        } else {
            categories[category] = [key];
        }
    });
    return categories;
}

function PolicyBuilderKeys({ keys, className }) {
    const [categories, setCategories] = useState(getKeysByCategory(keys));
    useEffect(() => {
        setCategories(getKeysByCategory(keys));
    }, [keys]);

    return (
        <div className={`flex flex-col px-3 pt-3 bg-primary-300 ${className}`}>
            <div className="-ml-6 -mr-3 bg-primary-500 p-2 rounded-bl rounded-tl text-base-100">
                Drag out a policy field
            </div>
            <div className="overflow-y-scroll">
                {Object.keys(categories).map((category, idx) => (
                    <CollapsibleSection
                        title={category}
                        key={category}
                        headerClassName="py-1"
                        titleClassName="w-full"
                        dataTestId="policy-key-group"
                        defaultOpen={idx === 0}
                    >
                        {categories[category].map((key) => {
                            return <PolicyBuilderKey key={key.name} fieldKey={key} />;
                        })}
                    </CollapsibleSection>
                ))}
            </div>
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
