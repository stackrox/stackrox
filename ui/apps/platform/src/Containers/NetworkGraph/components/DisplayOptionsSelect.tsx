import React, { useState } from 'react';
import {
    Select,
    SelectVariant,
    SelectGroup,
    SelectOption,
    Split,
    SplitItem,
} from '@patternfly/react-core';
import { PficonNetworkRangeIcon } from '@patternfly/react-icons';

import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';
import { ReactComponent as PortLabel } from 'images/network-graph/tcp-icon.svg';
import { ReactComponent as RelatedEntity } from 'images/network-graph/related-entity.svg';
import { ReactComponent as FilteredEntity } from 'images/network-graph/filtered-entity.svg';

import './DisplayOptionsSelect.css';
import { CidrBlockIcon, DeploymentIcon, NamespaceIcon } from '../common/NetworkGraphIcons';

export type DisplayOption =
    | 'policyStatusBadge'
    | 'externalBadge'
    | 'edgeLabel'
    | 'selectionIndicator'
    | 'objectTypeLabel';

type DisplayOptionsSelectProps = {
    selectedOptions: DisplayOption[];
    setSelectedOptions: (options) => void;
    isDisabled: boolean;
};

function DisplayOptionsSelect({
    selectedOptions,
    setSelectedOptions,
    isDisabled,
}: DisplayOptionsSelectProps) {
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
            toggleAriaLabel="Select display options"
            isDisabled={isDisabled}
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
                    <PortLabel width="22px" height="22px" className="pf-u-mr-xs" />
                    Port and protocol label
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Selection indicators" key="selection-indicator">
                <SelectOption key={2} value="selectionIndicator">
                    <Split>
                        <SplitItem className="pf-u-mr-xs">
                            <FilteredEntity width="24px" height="24px" />
                        </SplitItem>
                        <SplitItem>Filtered</SplitItem>
                        <SplitItem className="pf-u-mx-sm">&</SplitItem>
                        <SplitItem className="pf-u-mr-xs">
                            <RelatedEntity width="18px" height="18px" />
                        </SplitItem>
                        <SplitItem>Related entities</SplitItem>
                    </Split>
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Object type labels" key="object-type-labels">
                <SelectOption key={2} value="objectTypeLabel">
                    <Split>
                        <SplitItem className="pf-u-mr-xs">
                            <NamespaceIcon screenReaderText="namespace" />
                        </SplitItem>
                        <SplitItem className="pf-u-mr-xs">
                            <DeploymentIcon screenReaderText="deployment" />
                        </SplitItem>
                        <SplitItem className="pf-u-mr-xs">
                            <CidrBlockIcon screenReaderText="cidr block" />
                        </SplitItem>
                        <SplitItem>Labels</SplitItem>
                    </Split>
                </SelectOption>
            </SelectGroup>
        </Select>
    );
}

export default DisplayOptionsSelect;
