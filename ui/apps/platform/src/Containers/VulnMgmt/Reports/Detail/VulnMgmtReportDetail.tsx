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
import { vulnManagementReportsPath } from 'routePaths';
import { ReportConfiguration } from 'types/report.proto';

// import ReportOverview from './ReportOverview';

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

    function onEditReport() {
        history.replace({
            pathname: `${vulnManagementReportsPath}/${id}`,
            search: 'action=edit',
        });
    }

    return (
        <>
            <Breadcrumb className="pf-u-mb-md">
                <BreadcrumbItemLink to={vulnManagementReportsPath}>Policies</BreadcrumbItemLink>
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
                                    key="Edit report"
                                    component="button"
                                    onClick={onEditReport}
                                >
                                    Edit report
                                </DropdownItem>,
                            ]}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </>
    );
}

export default VulnMgmtReportDetail;
