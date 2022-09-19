import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbHeading,
    Dropdown,
    BadgeToggle,
    DropdownItem,
    Select,
    SelectOption,
    Spinner,
} from '@patternfly/react-core';
import { AngleLeftIcon } from '@patternfly/react-icons';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Cluster } from 'types/cluster.proto';

type EntityBreadcrumbsProps = {
    id?: string;
    setSelectedClusterId: (clusterId: string) => void;
    clusters: Cluster[];
    selectedClusterId?: string;
    isDisabled?: boolean;
    isLoading?: boolean;
    error?: string;
};

const EntityBreadcrumbs = ({
    id,
    setSelectedClusterId,
    clusters,
    selectedClusterId = '',
    isDisabled = false,
    isLoading = false,
    error = '',
}: EntityBreadcrumbsProps): ReactElement => {
    // const { closeSelect, isOpen, onToggle } = useSelectToggle();
    // function changeCluster(_e, clusterId) {
    //     setSelectedClusterId(clusterId);
    //     closeSelect();
    // }

    const dropdownItems: JSX.Element[] = [
        <DropdownItem key="edit" component="button" icon={<AngleLeftIcon />}>
            Edit
        </DropdownItem>,
        <DropdownItem key="action" component="button" icon={<AngleLeftIcon />}>
            Deployment
        </DropdownItem>,
        <DropdownItem key="apps" component="button" icon={<AngleLeftIcon />}>
            Applications
        </DropdownItem>,
    ];
    const [isOpen, setIsOpen] = React.useState(false);
    const badgeToggleRef = React.useRef<HTMLButtonElement>(null);

    const onToggle = (isOpen: boolean) => setIsOpen(isOpen);

    const onSelect = () => {
        setIsOpen((prevIsOpen: boolean) => !prevIsOpen);
        if (badgeToggleRef?.current) {
            badgeToggleRef.current.focus();
        }
    };

    return (
        <Breadcrumb>
            <BreadcrumbItem component="button">Section home</BreadcrumbItem>
            <BreadcrumbItem isDropdown>
                <Dropdown
                    onSelect={onSelect}
                    toggle={
                        <BadgeToggle ref={badgeToggleRef} onToggle={onToggle}>
                            {dropdownItems.length}
                        </BadgeToggle>
                    }
                    isOpen={isOpen}
                    dropdownItems={dropdownItems}
                />
            </BreadcrumbItem>
            <BreadcrumbHeading component="button">Section title</BreadcrumbHeading>
        </Breadcrumb>
    );

    // return (
    //     <Select
    //         id={id}
    //         isOpen={isOpen}
    //         onToggle={onToggle}
    //         isDisabled={isDisabled || !!error || !clusters.length}
    //         selections={selectedClusterId}
    //         placeholderText={
    //             isLoading ? (
    //                 <Spinner isSVG size="sm" aria-label="Contents of the small example" />
    //             ) : (
    //                 'Select a cluster'
    //             )
    //         }
    //         onSelect={changeCluster}
    //     >
    //         {clusters.map(({ id: clusterId, name }) => (
    //             <SelectOption key={clusterId} value={clusterId}>
    //                 {name}
    //             </SelectOption>
    //         ))}
    //     </Select>
    // );
};

export default EntityBreadcrumbs;
