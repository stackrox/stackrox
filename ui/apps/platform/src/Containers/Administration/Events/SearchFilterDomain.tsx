import { Divider, SelectOption } from '@patternfly/react-core';
import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';

import { domains } from 'services/AdministrationEventsService';

const optionAll = 'All domains';

type SearchFilterDomainProps = {
    domain: string | undefined;
    isDisabled: boolean;
    setDomain: (domain: string | undefined) => void;
};

function SearchFilterDomain({ domain, isDisabled, setDomain }: SearchFilterDomainProps) {
    function onSelect(selection: string | number | undefined) {
        setDomain(selection === optionAll ? undefined : (selection as string | undefined));
    }

    const options = domains.map((domainArg) => (
        <SelectOption key={domainArg} value={domainArg}>
            {domainArg}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>
    );

    return (
        <SimpleSelect
            value={domain ?? optionAll}
            onChange={onSelect}
            isDisabled={isDisabled}
            ariaLabelMenu="Domain filter menu items"
            ariaLabelToggle="Domain filter menu toggle"
        >
            {options}
        </SimpleSelect>
    );
}

export default SearchFilterDomain;
