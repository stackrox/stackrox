import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbHeading,
    Dropdown,
    BadgeToggle,
    DropdownItem,
    Flex,
    FlexItem,
    PageSection,
} from '@patternfly/react-core';
import AngleLeftIcon from '@patternfly/react-icons/dist/esm/icons/angle-left-icon';

import './BreadcrumbPage.css';

const ClusterIcon: React.FC = () => <span className="co-m-resource-icon">CL</span>;
const NamespaceIcon: React.FC = () => (
    <span className="co-m-resource-icon co-m-resource-namespace">NS</span>
);
const DeploymentIcon: React.FC = () => (
    <span className="co-m-resource-icon co-m-resource-deployment">DE</span>
);

const clusterDropdownItems: JSX.Element[] = [
    <DropdownItem key="production" component="button" icon={<ClusterIcon />}>
        production
    </DropdownItem>,
    <DropdownItem key="security" component="button" icon={<ClusterIcon />}>
        Security
    </DropdownItem>,
];

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

function BreadcrumbPage() {
    const [isOpen, setIsOpen] = React.useState(false);
    const [isClusterOpen, setIsClusterOpen] = React.useState(false);
    const clusterToggleRef = React.useRef<HTMLButtonElement>(null);
    const badgeToggleRef = React.useRef<HTMLButtonElement>(null);

    const onClusterToggle = (newIsClusterOpen: boolean) => setIsClusterOpen(newIsClusterOpen);

    const onToggle = (newIsOpen: boolean) => setIsOpen(newIsOpen);

    const onClusterDropdownSelect = () => {
        setIsClusterOpen((prevIsOpen: boolean) => !prevIsOpen);
        clusterToggleRef?.current?.focus();
    };

    const onSelect = () => {
        setIsOpen((prevIsOpen: boolean) => !prevIsOpen);
        badgeToggleRef?.current?.focus();
    };
    return (
        <PageSection variant="light" isFilled id="policies-table-loading">
            <div>
                <Breadcrumb>
                    <BreadcrumbItem component="button">
                        <ClusterIcon />
                        <Dropdown
                            onSelect={onClusterDropdownSelect}
                            toggle={
                                <BadgeToggle ref={clusterToggleRef} onToggle={onClusterToggle}>
                                    Select a cluster {clusterDropdownItems.length}
                                </BadgeToggle>
                            }
                            isOpen={isClusterOpen}
                            dropdownItems={clusterDropdownItems}
                        />
                    </BreadcrumbItem>
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
            </div>
            <Flex>
                <FlexItem>
                    <span>
                        <ClusterIcon />
                        cluster
                    </span>
                </FlexItem>
                <FlexItem>
                    <span>
                        <NamespaceIcon />
                        namespace
                    </span>
                </FlexItem>
                <FlexItem>
                    <span>
                        <DeploymentIcon />
                        namespace
                    </span>
                </FlexItem>
            </Flex>
            <h1>Breadcrumb Example</h1>
        </PageSection>
    );
}

export default BreadcrumbPage;
