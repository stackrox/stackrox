import React, { useState } from 'react';
import { Select, SelectVariant, SelectGroup, SelectOption } from '@patternfly/react-core';
import { PficonNetworkRangeIcon } from '@patternfly/react-icons';

import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';

import './DisplayOptionsSelect.css';

export type DisplayOption = 'policyStatusBadge' | 'externalBadge' | 'edgeLabel';

type DisplayOptionsSelectProps = {
    selectedOptions: DisplayOption[];
    setSelectedOptions: (options) => void;
};

function DisplayOptionsSelect({ selectedOptions, setSelectedOptions }: DisplayOptionsSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onSelect(e, selection) {
        if (selectedOptions.includes(selection)) {
            setSelectedOptions(selectedOptions.filter((item) => item !== selection));
        } else {
            setSelectedOptions([...selectedOptions, selection]);
        }
    }

    return (
        <Select
            variant={SelectVariant.checkbox}
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            selections={selectedOptions}
            placeholderText="Display options"
            isGrouped
            id="display-options-dropdown"
        >
            <SelectGroup label="Deployment visuals" key="deployment">
                <SelectOption key={0} value="policyStatusBadge">
                    <NoPolicyRules width="22px" height="22px" className="pf-u-mr-xs" />
                    Network policy status badge
                </SelectOption>
                <SelectOption key={1} value="externalBadge">
                    <PficonNetworkRangeIcon className="pf-u-ml-xs pf-u-mr-sm" /> Active external
                    traffic badge
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Edge visuals" key="edge">
                <SelectOption key={2} value="edgeLabel">
                    Port and protocol label
                </SelectOption>
            </SelectGroup>
        </Select>
    );
}

export default DisplayOptionsSelect;
