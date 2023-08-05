import React, { ReactElement, useState } from 'react';
import { useHistory, useParams, generatePath } from 'react-router-dom';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Spinner,
    Dropdown,
    DropdownToggle,
    DropdownItem,
    DropdownSeparator,
    Tabs,
    Tab,
    TabTitleText,
    TabTitleIcon,
    TabAction,
    Popover,
} from '@patternfly/react-core';
import { CaretDownIcon, ClipboardCheckIcon, HelpIcon, HomeIcon } from '@patternfly/react-icons';

import { vulnerabilityReportsPath } from 'routePaths';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import useFetchReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReport';
import useDeleteModal from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteModal';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';

import { vulnerabilityReportPath } from '../pathsForVulnerabilityReporting';
import DeleteReportModal from '../components/DeleteReportModal';
import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import ReportJobs from './ReportJobs';

export type TabTitleProps = {
    icon?: ReactElement;
    children: string;
};

function TabTitle({ icon, children }: TabTitleProps): ReactElement {
    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {icon && (
                <FlexItem>
                    <TabTitleIcon>{icon}</TabTitleIcon>
                </FlexItem>
            )}
            <FlexItem>
                <TabTitleText>{children}</TabTitleText>
            </FlexItem>
        </Flex>
    );
}

function ViewVulnReportPage() {
    const history = useHistory();
    const { reportId } = useParams();
    const [isActionsDropdownOpen, setIsActionsDropdownOpen] = useState(false);

    const { reportConfiguration, isLoading, error } = useFetchReport(reportId);

    const {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteError,
    } = useDeleteModal({
        onCompleted: () => {
            history.push(vulnerabilityReportsPath);
        },
    });

    function onToggleActionsDropdown() {
        setIsActionsDropdownOpen((prevValue) => !prevValue);
    }

    function onSelectAction() {
        setIsActionsDropdownOpen(false);
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (error || !reportConfiguration) {
        return (
            <NotFoundMessage
                title="Error fetching the report configuration"
                message={error || 'No data available'}
                actionText="Go to reports"
                url={vulnerabilityReportsPath}
            />
        );
    }

    const vulnReportPageURL = generatePath(vulnerabilityReportPath, {
        reportId: reportConfiguration.id,
    }) as string;

    const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);

    return (
        <>
            <PageTitle title="View vulnerability report" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{reportConfiguration.name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'row' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">{reportConfiguration.name}</Title>
                    </FlexItem>
                    <FlexItem>
                        <Dropdown
                            onSelect={onSelectAction}
                            position="right"
                            toggle={
                                <DropdownToggle
                                    isPrimary
                                    onToggle={onToggleActionsDropdown}
                                    toggleIndicator={CaretDownIcon}
                                >
                                    Actions
                                </DropdownToggle>
                            }
                            isOpen={isActionsDropdownOpen}
                            dropdownItems={[
                                <DropdownItem
                                    key="Edit report"
                                    component="button"
                                    onClick={() => {
                                        history.push(`${vulnReportPageURL}?action=edit`);
                                    }}
                                >
                                    Edit report
                                </DropdownItem>,
                                <DropdownSeparator key="separator" />,
                                <DropdownItem
                                    key="Send report now"
                                    component="button"
                                    onClick={() => {}}
                                >
                                    Send report now
                                </DropdownItem>,
                                <DropdownItem
                                    key="Generate download"
                                    component="button"
                                    onClick={() => {}}
                                >
                                    Generate download
                                </DropdownItem>,
                                <DropdownItem
                                    key="Clone report"
                                    component="button"
                                    onClick={() => {
                                        history.push(`${vulnReportPageURL}?action=clone`);
                                    }}
                                >
                                    Clone report
                                </DropdownItem>,
                                <DropdownSeparator key="Separator" />,
                                <DropdownItem
                                    key="Delete report"
                                    className="pf-u-danger-color-100"
                                    component="button"
                                    onClick={() => {
                                        openDeleteModal(reportConfiguration.id);
                                    }}
                                >
                                    Delete report
                                </DropdownItem>,
                            ]}
                        />
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <Tabs
                    className="pf-u-background-color-100"
                    defaultActiveKey={0}
                    aria-label="Report details tabs"
                    role="region"
                >
                    <Tab
                        eventKey={0}
                        title={<TabTitle icon={<HomeIcon />}>Configuration details</TabTitle>}
                        aria-label="Configuration details tab"
                    >
                        <PageSection
                            variant="light"
                            padding={{ default: 'noPadding' }}
                            className="pf-u-py-lg pf-u-px-lg"
                        >
                            <ReportParametersDetails formValues={reportFormValues} />
                            <Divider component="div" className="pf-u-py-md" />
                            <DeliveryDestinationsDetails formValues={reportFormValues} />
                            <Divider component="div" className="pf-u-py-md" />
                            <ScheduleDetails formValues={reportFormValues} />
                        </PageSection>
                    </Tab>
                    <Tab
                        eventKey={1}
                        title={<TabTitle icon={<ClipboardCheckIcon />}>All report jobs</TabTitle>}
                        aria-label="Report jobs tab"
                        actions={
                            <>
                                <Popover
                                    aria-label="All report jobs help text"
                                    headerContent={<div>All report jobs</div>}
                                    bodyContent={
                                        <div>
                                            This function displays the requested jobs from different
                                            users and includes their statuses accordingly. While the
                                            function provides the ability to monitor and audit your
                                            active and past requested jobs, we suggest configuring
                                            the &quot;Vulnerability report retention limit&quot;
                                            based on your needs in order to ensure optimal user
                                            experience. All the report jobs will be kept in your
                                            system until they exceed the limit set by you.
                                        </div>
                                    }
                                    enableFlip
                                    position="top"
                                >
                                    <TabAction aria-label="Help for report jobs tab">
                                        <HelpIcon />
                                    </TabAction>
                                </Popover>
                            </>
                        }
                    >
                        <PageSection
                            padding={{ default: 'noPadding' }}
                            className="pf-u-py-lg pf-u-px-lg"
                        >
                            <ReportJobs reportId={reportId} />
                        </PageSection>
                    </Tab>
                </Tabs>
            </PageSection>
            <DeleteReportModal
                isOpen={isDeleteModalOpen}
                onClose={closeDeleteModal}
                isDeleting={isDeleting}
                onDelete={onDelete}
                error={deleteError}
            />
        </>
    );
}

export default ViewVulnReportPage;
