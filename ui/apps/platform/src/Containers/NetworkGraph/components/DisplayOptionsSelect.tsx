import React from 'react';
import { Split, SplitItem, SelectGroup, SelectOption } from '@patternfly/react-core';
import { PficonNetworkRangeIcon } from '@patternfly/react-icons';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';

import NoPolicyRules from 'images/network-graph/no-policy-rules.svg?react';
import PortLabel from 'images/network-graph/tcp-icon.svg?react';
import RelatedEntity from 'images/network-graph/related-entity.svg?react';
import FilteredEntity from 'images/network-graph/filtered-entity.svg?react';

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
    setSelectedOptions: (options: DisplayOption[]) => void;
    isDisabled: boolean;
};

function DisplayOptionsSelect({
    selectedOptions,
    setSelectedOptions,
    isDisabled,
}: DisplayOptionsSelectProps) {
    function handleChange(selections: string[]) {
        setSelectedOptions(selections as DisplayOption[]);
    }

    return (
        <CheckboxSelect
            id="display-options-dropdown"
            selections={selectedOptions}
            onChange={handleChange}
            ariaLabel="Select display options"
            placeholderText="Display options"
            isDisabled={isDisabled}
        >
            <SelectGroup label="Deployment visuals" key="deployment">
                <SelectOption key={0} value="policyStatusBadge">
                    <NoPolicyRules width="22px" height="22px" className="pf-v5-u-mr-xs" />
                    Network policy status badge
                </SelectOption>
                <SelectOption key={1} value="externalBadge">
                    <PficonNetworkRangeIcon className="pf-v5-u-ml-xs pf-v5-u-mr-sm" /> Active
                    external traffic badge
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Edge visuals" key="edge">
                <SelectOption key={2} value="edgeLabel">
                    <PortLabel width="22px" height="22px" className="pf-v5-u-mr-xs" />
                    Port and protocol label
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Selection indicators" key="selection-indicator">
                <SelectOption key={3} value="selectionIndicator">
                    <Split>
                        <SplitItem className="pf-v5-u-mr-xs">
                            <FilteredEntity width="24px" height="24px" />
                        </SplitItem>
                        <SplitItem>Filtered</SplitItem>
                        <SplitItem className="pf-v5-u-mx-sm">&</SplitItem>
                        <SplitItem className="pf-v5-u-mr-xs">
                            <RelatedEntity width="18px" height="18px" />
                        </SplitItem>
                        <SplitItem>Related entities</SplitItem>
                    </Split>
                </SelectOption>
            </SelectGroup>
            <SelectGroup label="Object type labels" key="object-type-labels">
                <SelectOption key={4} value="objectTypeLabel">
                    <Split>
                        <SplitItem className="pf-v5-u-mr-xs">
                            <NamespaceIcon screenReaderText="namespace" />
                        </SplitItem>
                        <SplitItem className="pf-v5-u-mr-xs">
                            <DeploymentIcon screenReaderText="deployment" />
                        </SplitItem>
                        <SplitItem className="pf-v5-u-mr-xs">
                            <CidrBlockIcon screenReaderText="cidr block" />
                        </SplitItem>
                        <SplitItem>Labels</SplitItem>
                    </Split>
                </SelectOption>
            </SelectGroup>
        </CheckboxSelect>
    );
}

export default DisplayOptionsSelect;
