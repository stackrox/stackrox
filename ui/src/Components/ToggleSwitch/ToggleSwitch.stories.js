/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';
import { Power } from 'react-feather';

import ToggleSwitch from './ToggleSwitch';

export default {
    title: 'ToggleSwitch',
    component: ToggleSwitch,
};

export const withSwitchOnly = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return <ToggleSwitch id="withSwitchOnly" toggleHandler={handleChange} enabled={isToggled} />;
};

export const withAlertClass = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return (
        <ToggleSwitch
            id="withAlertClass"
            toggleHandler={handleChange}
            enabled={isToggled}
            extraClassNames="toggle-switch-alert"
        />
    );
};

export const withSmallAttribute = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return (
        <ToggleSwitch
            id="withSmallAttribute"
            toggleHandler={handleChange}
            enabled={isToggled}
            small
        />
    );
};

export const withLabel = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return (
        <ToggleSwitch
            id="withLabel"
            toggleHandler={handleChange}
            label="Enable AutoUpgrade"
            enabled={isToggled}
        />
    );
};

export const withFormWrapperAndSeparateLabel = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return (
        <div className="mb-4 flex bg-base-100 border-2 rounded px-2 py-1 border-base-300 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10 border-base-300 items-center justify-between">
            <label htmlFor="admissionController" className="block py-2 text-base-600 font-700">
                Create Admission Controller Webhook
            </label>
            <ToggleSwitch
                id="admissionController"
                name="admissionController"
                toggleHandler={handleChange}
                enabled={isToggled}
            />
        </div>
    );
};

export const withFormWrapperSeparateLabelAndIcon = () => {
    const [isToggled, setIsToggled] = useState(false);

    function handleChange() {
        setIsToggled(!isToggled);
    }

    return (
        <div className="inline-flex bg-success-200 border-2 rounded px-2 py-1 border-success-600 font-600 text-success-600 hover:border-success-300 leading-normal items-center justify-between">
            <Power />
            <label
                htmlFor="enableDisablePolicy"
                className="block pl-2 pt-1 leading-none text-success-600 font-700"
            >
                POLICY
            </label>
            <ToggleSwitch
                id="enableDisablePolicy"
                name="enableDisablePolicy"
                toggleHandler={handleChange}
                enabled={isToggled}
            />
        </div>
    );
};
