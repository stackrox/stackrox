import { Divider, SelectOption } from '@patternfly/react-core';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

import { domains } from 'services/AdministrationEventsService';

const optionAll = 'All domains';

type SearchFilterDomainProps = {
    domain: string | undefined;
    isDisabled: boolean;
    setDomain: (domain: string | undefined) => void;
};

function SearchFilterDomain({ domain, isDisabled, setDomain }: SearchFilterDomainProps) {
    function onSelect(_id: string, selection: string) {
        setDomain(selection === optionAll ? undefined : selection);
    }

    const options = [
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>,
        <Divider key="Divider" />,
        ...domains.map((domainArg) => (
            <SelectOption key={domainArg} value={domainArg}>
                {domainArg}
            </SelectOption>
        )),
    ];

    return (
        <SelectSingle
            id="domain-filter"
            value={domain ?? optionAll}
            handleSelect={onSelect}
            isDisabled={isDisabled}
            placeholderText="Select domain"
            toggleAriaLabel="Domain filter menu toggle"
        >
            {options}
        </SelectSingle>
    );
}

export default SearchFilterDomain;
