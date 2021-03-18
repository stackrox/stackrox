import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { formValueSelector } from 'redux-form';
import { HelpIcon } from '@stackrox/ui-components';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import PolicyBuilderKey from 'Components/PolicyBuilderKey';
import CollapsibleSection from 'Components/CollapsibleSection';
import RadioButtonGroup from 'Components/RadioButtonGroup';
import { resourceTypes } from 'constants/entityTypes';

const entityFilterButtons = [
    {
        text: resourceTypes.DEPLOYMENT,
    },
    {
        text: resourceTypes.NODE,
    },
    {
        text: resourceTypes.CLUSTER,
    },
];

function getKeysByCategory(keys, k8sAuditLogEnabled) {
    const categories = {};
    if (k8sAuditLogEnabled) {
        // grouping keys by entity type, then category
        keys.forEach((key) => {
            const { category, entityType } = key;
            if (categories[entityType] && categories[entityType][category]) {
                // entity type and category already exist
                categories[entityType][category].push(key);
            } else if (categories[entityType]) {
                // entity type exists, but category does not exist yet
                categories[entityType][category] = [key];
            } else {
                // entity type nor category exist
                categories[entityType] = {
                    [category]: [key],
                };
            }
        });
        return categories;
    }

    // TODO: remove when k8s audit logging feature flag is removed
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

function getSelectedEntityFilter(firstFormValue, keys) {
    if (firstFormValue?.fieldName) {
        const fieldKey = keys.find((key) => key.name === firstFormValue.fieldName);
        return fieldKey.entityType;
    }
    return entityFilterButtons[0].text;
}

function PolicyBuilderKeys({ keys, className, firstFormValue }) {
    const k8sAuditLogEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION);
    const [selectedEntityFilter, setSelectedEntityFilter] = useState(
        getSelectedEntityFilter(firstFormValue, keys)
    );
    const [allCategories] = useState(getKeysByCategory(keys, k8sAuditLogEnabled));
    const [filteredCategories, setFilteredCategories] = useState(
        k8sAuditLogEnabled ? allCategories[selectedEntityFilter] : allCategories
    );

    function handleEntityFilterToggle(entity) {
        setSelectedEntityFilter(entity);
        setFilteredCategories(allCategories[entity]);
    }

    return (
        <div className={`flex flex-col px-3 pt-3 bg-primary-300 ${className}`}>
            <div className="-ml-6 -mr-3 bg-primary-500 p-2 rounded-bl rounded-tl text-base-100">
                Drag out a policy field
            </div>
            {k8sAuditLogEnabled && (
                <div className="flex py-2">
                    <RadioButtonGroup
                        buttons={entityFilterButtons}
                        onClick={handleEntityFilterToggle}
                        selected={selectedEntityFilter}
                        disabled={firstFormValue?.fieldName}
                        testId="policy-key-filter"
                    />
                    <div className="ml-2 flex items-center">
                        <HelpIcon description="This field is disabled when using entity specific criteria. Clear all criteria to change entities." />
                    </div>
                </div>
            )}
            <div className="overflow-y-scroll">
                {Object.keys(filteredCategories).map((category, idx) => (
                    <CollapsibleSection
                        title={category}
                        key={category}
                        headerClassName="py-1"
                        titleClassName="w-full"
                        dataTestId="policy-key-group"
                        defaultOpen={idx === 0}
                    >
                        {filteredCategories[category].map((key) => {
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
    firstFormValue: PropTypes.shape({
        fieldName: PropTypes.string,
    }),
};

PolicyBuilderKeys.defaultProps = {
    className: 'w-1/3',
    firstFormValue: {},
};

const mapStateToProps = createStructuredSelector({
    firstFormValue: (state) =>
        formValueSelector('policyCreationForm')(state, 'policySections[0].policyGroups[0]'),
});

export default connect(mapStateToProps)(PolicyBuilderKeys);
