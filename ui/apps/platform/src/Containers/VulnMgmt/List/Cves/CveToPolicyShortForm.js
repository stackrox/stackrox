import React from 'react';
import PropTypes from 'prop-types';

import MultiSelect from 'Components/MultiSelect';
import ReactSelect, { Creatable } from 'Components/ReactSelect';
import ToggleSwitch from 'Components/ToggleSwitch';

const severityOptions = [
    {
        label: 'Critical',
        value: 'CRITICAL_SEVERITY',
    },
    {
        label: 'High',
        value: 'HIGH_SEVERITY',
    },
    {
        label: 'Medium',
        value: 'MEDIUM_SEVERITY',
    },
    {
        label: 'Low',
        value: 'LOW_SEVERITY',
    },
];

const lifecycleOptions = [
    {
        label: 'Build',
        value: 'BUILD',
    },
    {
        label: 'Deploy',
        value: 'DEPLOY',
    },
    // no RUNTIME enforcement for policies based on CVEs
];

export const emptyPolicy = {
    name: '',
    severity: '',
    lifecycleStages: [],
    description: '',
    disabled: false,
    categories: ['Vulnerability Management'],
    policySections: [],
    exclusions: [],
};

function wrapSelectEvent(key, handleChange) {
    return function compareSelected(selectedOption) {
        const syntheticEvent = {
            target: {
                name: key,
                value: selectedOption,
            },
        };

        handleChange(syntheticEvent);
    };
}

function CveToPolicyShortForm({
    policy,
    handleChange,
    policies,
    selectedPolicy,
    setSelectedPolicy,
}) {
    // curry the change handlers for the select inputs
    const onSeverityChange = wrapSelectEvent('severity', handleChange);
    const onLifeCycleChange = wrapSelectEvent('lifecycleStages', handleChange);

    // values for accessibilty and testing selector hooks
    const identifierForNameField = 'policy-name-to-use';
    const identifierForSeverityField = 'severity-to-use';
    const identifierForLifecycleField = 'lifecycle-to-use';

    function createNewOption(policyName) {
        const newPolicy = {
            ...emptyPolicy,
            name: policyName,
            label: policyName,
            value: policies.length,
        };
        setSelectedPolicy(newPolicy);
    }

    function onChange(idx) {
        setSelectedPolicy(policies[idx]);
    }

    return (
        <form className="w-full mb-4" data-testid="policy-short-form">
            <div className="mb-4">
                <label
                    htmlFor={identifierForNameField}
                    className="block py-2 text-base-600 font-700"
                >
                    Policy Name{' '}
                    <span
                        aria-label="Required"
                        data-testid="required"
                        className="text-alert-500 ml-1"
                    >
                        *
                    </span>
                </label>
                <div className="flex">
                    <Creatable
                        key="policy"
                        onChange={onChange}
                        onCreateOption={createNewOption}
                        options={policies}
                        placeholder="Type a name, or select an existing policy"
                        value={selectedPolicy}
                        data-testid={identifierForNameField}
                        allowCreateWhileLoading
                    />
                </div>
                <div className="p-2 text-sm font-500">
                    <em>(Names for new policies must be at least 6 characters.)</em>
                </div>
            </div>
            <div className="mb-4 flex justify-between">
                <div className="flex flex-col w-full mr-1">
                    <label
                        htmlFor={identifierForSeverityField}
                        className="block py-2 text-base-600 font-700"
                    >
                        Severity{' '}
                        <span
                            aria-label="Required"
                            data-testid="required"
                            className="text-alert-500 ml-1"
                        >
                            *
                        </span>
                    </label>
                    <div className="flex">
                        <ReactSelect
                            id="severity"
                            name="severity"
                            options={severityOptions}
                            placeholder="Select severity"
                            onChange={onSeverityChange}
                            className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                            wrapperClass="bg-base-100 border-2 rounded border-base-300 w-full text-base-600 hover:border-base-400"
                            triggerClass="border-l border-base-300"
                            value={policy.severity}
                            disabled={!!policy.id}
                            data-testid={identifierForSeverityField}
                        />
                    </div>
                </div>
                <div className="flex flex-col w-full ml-1">
                    <label
                        htmlFor={identifierForLifecycleField}
                        className="block py-2 text-base-600 font-700"
                    >
                        Lifecycle Stage{' '}
                        <span
                            aria-label="Required"
                            data-testid="required"
                            className="text-alert-500 ml-1"
                        >
                            *
                        </span>
                    </label>
                    <div className="flex">
                        <MultiSelect
                            id="lifecycleStages"
                            name="lifecycleStages"
                            options={lifecycleOptions}
                            placeholder="Select Lifecycle Stages"
                            onChange={onLifeCycleChange}
                            className="block w-full bg-base-100 border-base-300 text-base-600 z-1 focus:border-base-500"
                            value={policy.lifecycleStages}
                            disabled={!!policy.id}
                            data-testid={identifierForLifecycleField}
                        />
                    </div>
                </div>
            </div>

            <div className="mb-4">
                <label htmlFor="description" className="block py-2 text-base-600 font-700">
                    Description
                </label>
                <div className="flex">
                    <textarea
                        id="description"
                        name="description"
                        value={policy.description}
                        onChange={handleChange}
                        disabled={!!policy.id}
                        placeholder="What does this policy do?"
                        className="bg-base-100 border-2 rounded p-2 border-base-300 w-full text-base-600 hover:border-base-400 leading-normal min-h-32"
                    />
                </div>
            </div>
            <div className="mb-4">
                <div className="flex items-center min-w-64 ml-4">
                    <ToggleSwitch
                        id="disabled"
                        name="disabled"
                        toggleHandler={handleChange}
                        label="Enabled"
                        enabled={policy.disabled}
                        flipped
                        small
                    />
                </div>
            </div>
        </form>
    );
}

CveToPolicyShortForm.propTypes = {
    policy: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string,
        severity: PropTypes.string,
        lifecycleStages: PropTypes.arrayOf(
            PropTypes.shape({
                label: PropTypes.string,
                value: PropTypes.string,
            })
        ),
        description: PropTypes.string,
        disabled: PropTypes.bool,
        categories: PropTypes.arrayOf(PropTypes.string),
        fields: PropTypes.shape({
            cve: PropTypes.string,
        }),
        exclusions: PropTypes.arrayOf(PropTypes.shape({})),
    }).isRequired,
    policies: PropTypes.arrayOf(
        PropTypes.shape({ label: PropTypes.string, value: PropTypes.string })
    ),
    selectedPolicy: PropTypes.number,
    setSelectedPolicy: PropTypes.func.isRequired,
    handleChange: PropTypes.func.isRequired,
};

CveToPolicyShortForm.defaultProps = {
    policies: [],
    selectedPolicy: null,
};

export default CveToPolicyShortForm;
