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
} from '@patternfly/react-core';
import { CaretDownIcon, DownloadIcon, HistoryIcon, HomeIcon } from '@patternfly/react-icons';

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
import RunHistory from './RunHistory';

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
                        title={<TabTitle icon={<HistoryIcon />}>Run history</TabTitle>}
                        aria-label="Run history tab"
                    >
                        <PageSection
                            padding={{ default: 'noPadding' }}
                            className="pf-u-py-lg pf-u-px-lg"
                        >
                            <RunHistory reportId={reportId} />
                        </PageSection>
                    </Tab>
                    <Tab
                        eventKey={2}
                        title={<TabTitle icon={<DownloadIcon />}>Downloadable report</TabTitle>}
                        aria-label="Downloadable report tab"
                    >
                        <div />
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
