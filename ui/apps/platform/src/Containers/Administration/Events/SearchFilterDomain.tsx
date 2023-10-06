import React, { useState } from 'react';
import { Divider, Select, SelectOption } from '@patternfly/react-core';

import { domains } from 'services/AdministrationEventsService';

const optionAll = 'All';

type SearchFilterDomainProps = {
    domain: string | undefined;
    setDomain: (domain: string | undefined) => void;
};

function SearchFilterDomain({ domain, setDomain }: SearchFilterDomainProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle(isOpenArg: boolean) {
        setIsOpen(isOpenArg);
    }

    function onSelect(_event, selection) {
        setDomain(selection === optionAll ? undefined : selection);
        setIsOpen(false);
    }

    const options = domains.map((domainArg) => (
        <SelectOption key={domainArg} value={domainArg}>
            {domainArg}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll} isPlaceholder>
            All domains
        </SelectOption>
    );

    return (
        <Select
            variant="single"
            aria-label="Domain filter menu items"
            toggleAriaLabel="Domain filter menu toggle"
            onToggle={onToggle}
            onSelect={onSelect}
            selections={domain ?? optionAll}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterDomain;
