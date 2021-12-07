import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Breadcrumb,
    BreadcrumbItem,
    Dropdown,
    DropdownItem,
    DropdownToggle,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';
import { ReportConfiguration } from 'types/report.proto';

// import PolicyOverview from './PolicyOverview';

type VulnMgmtReportDetailProps = {
    report: ReportConfiguration;
};

function VulnMgmtReportDetail({ report }: VulnMgmtReportDetailProps): ReactElement {
    const history = useHistory();

    const [isActionsOpen, setIsActionsOpen] = useState(false);

    const { id, name } = report;

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onToggleActions(isOpen) {
        setIsActionsOpen(isOpen);
    }

    function onEditPolicy() {
        history.replace({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onClonePolicy() {
        history.replace({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=clone',
        });
    }

    return (
        <>
            <Breadcrumb className="pf-u-mb-md">
                <BreadcrumbItemLink to={policiesBasePath}>Policies</BreadcrumbItemLink>
                <BreadcrumbItem isActive>{name}</BreadcrumbItem>
            </Breadcrumb>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h1">{name}</Title>
                    </ToolbarItem>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Dropdown
                            onSelect={onSelectActions}
                            position="right"
                            toggle={
                                <DropdownToggle
                                    isPrimary
                                    onToggle={onToggleActions}
                                    toggleIndicator={CaretDownIcon}
                                >
                                    Actions
                                </DropdownToggle>
                            }
                            isOpen={isActionsOpen}
                            dropdownItems={[
                                <DropdownItem
                                    key="Edit policy"
                                    component="button"
                                    onClick={onEditPolicy}
                                >
                                    Edit policy
                                </DropdownItem>,
                                <DropdownItem
                                    key="Clone policy"
                                    component="button"
                                    onClick={onClonePolicy}
                                >
                                    Clone policy
                                </DropdownItem>,
                            ]}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Title headingLevel="h2" className="pf-u-mb-md">
                Policy overview
            </Title>
            <Title headingLevel="h2" className="pf-u-mb-md">
                Policy criteria
            </Title>
        </>
    );
}

export default VulnMgmtReportDetail;
