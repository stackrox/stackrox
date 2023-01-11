import React, { useState } from 'react';
import {
    Select,
    SelectVariant,
    SelectGroup,
    SelectOption,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { PficonNetworkRangeIcon } from '@patternfly/react-icons';

import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';

type DisplayOptionsSelectProps = {
    selectedOptions: string[];
    setSelectedOptions: (options) => void;
};

function DisplayOptionsSelect({ selectedOptions, setSelectedOptions }: DisplayOptionsSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    console.log(selectedOptions);

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
        >
            <SelectGroup label="Deployment visuals" key="deployment">
                <SelectOption key={0} value="policyStatusBadge">
                    <Flex>
                        <FlexItem>
                            <NoPolicyRules width="22px" height="22px" />
                        </FlexItem>
                        <FlexItem>
                            <span>Network policy status badge</span>
                        </FlexItem>
                    </Flex>
                </SelectOption>
                <SelectOption key={1} value="externalBadge">
                    <PficonNetworkRangeIcon /> Active external traffic badge
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
