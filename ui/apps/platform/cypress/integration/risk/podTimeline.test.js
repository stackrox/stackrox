import { format } from 'date-fns';

import { selectors, url } from '../../constants/RiskPage';

import * as api from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

function setRoutes() {
    cy.intercept('GET', api.risks.riskyDeployments).as('deployments');
    cy.intercept('GET', api.risks.fetchDeploymentWithRisk).as('getDeployment');
    cy.intercept('POST', api.graphql(api.risks.graphqlOps.getDeploymentEventTimeline)).as(
        'getDeploymentEventTimeline'
    );
    cy.intercept('POST', api.graphql(api.risks.graphqlOps.getPodEventTimeline)).as(
        'getPodEventTimeline'
    );
}

function openDeployment(deploymentName) {
    cy.visit(url);
    cy.wait('@deployments');

    cy.get(`${selectors.table.rows}:contains(${deploymentName})`).click();
    cy.wait('@getDeployment');
}

function openEventTimeline() {
    openDeployment('collector');
    // open the process discovery tab
    cy.get(selectors.sidePanel.processDiscoveryTab).click();
    cy.get(selectors.eventTimelineOverviewButton).click();
}

// mocks the graphql operation name "getPodEventTimeline" and drills down to the pod->containers view
function openMockedPodEventTimelineView() {
    setRoutes();
    // mocking data to thoroughly test the event details
    cy.intercept('POST', api.graphql(api.risks.graphqlOps.getPodEventTimeline), {
        fixture: 'risks/eventTimeline/podEventTimeline.json',
    }).as('getPodEventTimeline');

    openEventTimeline();

    cy.wait('@getDeploymentEventTimeline');

    // click the button and drill down to see containers
    cy.get(selectors.eventTimeline.timeline.namesList.drillDownButtonInFirstRow).click();

    cy.wait('@getPodEventTimeline');
}

describe('Risk Page Pod Event Timeline', () => {
    withAuth();

    describe('Clustering Events', () => {
        it('should show the clustered event markers', () => {
            setRoutes();
            // mocking data to thoroughly test the clustering
            cy.intercept('POST', api.graphql(api.risks.graphqlOps.getPodEventTimeline), {
                fixture: 'risks/eventTimeline/clusteredPodEventTimeline.json',
            }).as('getPodEventTimeline');

            openEventTimeline();

            cy.wait('@getDeploymentEventTimeline');

            // click the button and drill down to see containers
            cy.get(selectors.eventTimeline.timeline.namesList.drillDownButtonInFirstRow).click();

            cy.wait('@getPodEventTimeline');

            cy.get(selectors.eventTimeline.timeline.mainView.allClusteredEvents).should(
                'have.length',
                2
            );
            // there should be a clustered event for events of different types
            cy.get(selectors.eventTimeline.timeline.mainView.clusteredEvent.generic).should(
                'exist'
            );
            // there should be a clustered event for events of the same type
            cy.get(selectors.eventTimeline.timeline.mainView.clusteredEvent.processActivity).should(
                'exist'
            );
        });

        it('should show the clustered event tooltip', () => {
            setRoutes();
            // mocking data to thoroughly test the filtering
            cy.intercept('POST', api.graphql(api.risks.graphqlOps.getPodEventTimeline), {
                fixture: 'risks/eventTimeline/clusteredPodEventTimeline.json',
            }).as('getPodEventTimeline');

            openEventTimeline();

            cy.wait('@getDeploymentEventTimeline');

            // click the button and drill down to see containers
            cy.get(selectors.eventTimeline.timeline.namesList.drillDownButtonInFirstRow).click();

            cy.wait('@getPodEventTimeline');

            cy.get(selectors.eventTimeline.timeline.mainView.clusteredEvent.generic).click();

            cy.get(selectors.tooltip.title).should('contain', '3 Events within 0 ms');
            cy.get(selectors.tooltip.bodyContent.eventDetails).should('have.length', 3);

            cy.get(
                selectors.eventTimeline.timeline.mainView.clusteredEvent.processActivity
            ).click();

            cy.get(selectors.tooltip.title).should('contain', '2 Events within 0 ms');
            cy.get(selectors.tooltip.bodyContent.eventDetails).should('have.length', 2);
        });
    });

    describe('Filtering Events By Type', () => {
        const FILTER_OPTIONS = {
            SHOW_ALL: 0,
            POLICY_VIOLATIONS: 1,
            PROCESS_ACTIVITIES: 2,
            RESTARTS: 3,
            TERMINATIONS: 4,
        };

        it('should filter policy violation events', () => {
            openMockedPodEventTimelineView();

            // the policy violation event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation);

            // filter by something else
            cy.get(selectors.eventTimeline.select.input).click();
            cy.get(
                `${selectors.eventTimeline.select.options}:eq(${FILTER_OPTIONS.PROCESS_ACTIVITIES})`
            ).click({ force: true });

            // the policy violation event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation).should(
                'not.exist'
            );
        });

        it('should filter process activity events and process in baseline activity events', () => {
            openMockedPodEventTimelineView();

            // the process activity + process in baseline activity event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.processActivity);
            cy.get(selectors.eventTimeline.timeline.mainView.event.processInBaselineActivity);

            // filter by something else
            cy.get(selectors.eventTimeline.select.input).click();
            cy.get(
                `${selectors.eventTimeline.select.options}:eq(${FILTER_OPTIONS.POLICY_VIOLATIONS})`
            ).click({ force: true });

            // the process activity + process in baseline activity event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.processActivity).should(
                'not.exist'
            );
            cy.get(
                selectors.eventTimeline.timeline.mainView.event.processInBaselineActivity
            ).should('not.exist');
        });

        it('should filter container restart events', () => {
            openMockedPodEventTimelineView();

            // the container restart event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart);

            // filter by something else
            cy.get(selectors.eventTimeline.select.input).click();
            cy.get(
                `${selectors.eventTimeline.select.options}:eq(${FILTER_OPTIONS.POLICY_VIOLATIONS})`
            ).click({ force: true });

            // thecontainer restart event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart).should('not.exist');
        });

        it('should filter container termination events', () => {
            openMockedPodEventTimelineView();

            // the container termination event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.termination);

            // filter by something else
            cy.get(selectors.eventTimeline.select.input).click();
            cy.get(
                `${selectors.eventTimeline.select.options}:eq(${FILTER_OPTIONS.POLICY_VIOLATIONS})`
            ).click({ force: true });

            // the container termination event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.containerTermination).should(
                'not.exist'
            );
        });
    });

    describe('Event Details', () => {
        /**
         * Finds an event based on the event id and returns the formatted timestamp
         * @param {string} id - the event id
         * @returns {Promise<string>} - a promise that, once resolved, will return the formatted timestamp of an event for the specified event typee
         */
        function getFormattedEventTimeById(id) {
            return cy.fixture('risks/eventTimeline/deploymentEventTimeline.json').then((json) => {
                const eventTime = json.data.pods[0].events.find(
                    (event) => event.id === id
                ).timestamp;
                return `Event time: ${format(eventTime, 'MM/DD/YYYY | h:mm:ssA')}`;
            });
        }

        it('shows the policy violation event details', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation).trigger(
                'mouseenter'
            );

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', 'Ubuntu Package Manager Execution');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Policy Violation');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('d7a275e1-1bba-47e7-92a1-42340c759883').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the process activity event details for a process with no parent', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(
                `${selectors.eventTimeline.timeline.mainView.event.processActivity}:eq(0)`
            ).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', '/usr/sbin/nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Process Activity');
            cy.get(selectors.tooltip.body).should('contain', 'Arguments: -g daemon off;');
            // if there's no parent process, then the text should display "No Parent"
            cy.get(selectors.tooltip.body).should('contain', 'Parent Name: No Parent');
            // if there's no parent process, then we shouln't display the parent uid
            cy.get(selectors.tooltip.body).should('not.contain', 'Parent UID: -1');
            cy.get(selectors.tooltip.body).should('contain', 'UID: 1000');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('e7519642-958a-534b-8297-59de4560d4ab').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the process activity event details for a process with a parent and unknown parent uid', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(
                `${selectors.eventTimeline.timeline.mainView.event.processActivity}:eq(1)`
            ).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', '/usr/sbin/nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Process Activity');
            cy.get(selectors.tooltip.body).should('contain', 'Arguments: -g daemon off;');
            cy.get(selectors.tooltip.body).should('contain', 'Parent Name: /usr/sbin/nginx');
            // if there's a parent process, and the parent uid is -1, it means that it's unknown
            cy.get(selectors.tooltip.body).should('contain', 'Parent UID: Unknown');
            cy.get(selectors.tooltip.body).should('contain', 'UID: 2000');
            cy.get(selectors.tooltip.bodyContent.uidFieldValue).should(
                'have.class',
                'text-alert-600'
            );
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('e7519642-958a-534b-8246-59de4560d4ab').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the process activity event details for a process with a uid change', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(
                `${selectors.eventTimeline.timeline.mainView.event.processActivity}:eq(2)`
            ).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', '/usr/sbin/nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Process Activity');
            cy.get(selectors.tooltip.body).should('contain', 'Arguments: -g daemon off;');
            cy.get(selectors.tooltip.body).should('contain', 'Parent Name: /usr/sbin/nginx');
            cy.get(selectors.tooltip.body).should('contain', 'Parent UID: 1000');
            cy.get(selectors.tooltip.body).should('contain', 'UID: 3000');
            cy.get(selectors.tooltip.bodyContent.uidFieldValue).should(
                'have.class',
                'text-alert-600'
            );
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('e7519642-958a-534b-8296-59de5560d4ab').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the process activity event details for a process with no uid change', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(
                `${selectors.eventTimeline.timeline.mainView.event.processActivity}:eq(3)`
            ).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', '/usr/sbin/nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Process Activity');
            cy.get(selectors.tooltip.body).should('contain', 'Arguments: -g daemon off;');
            cy.get(selectors.tooltip.body).should('contain', 'Parent Name: /usr/sbin/nginx');
            cy.get(selectors.tooltip.body).should('contain', 'Parent UID: 4000');
            cy.get(selectors.tooltip.body).should('contain', 'UID: 4000');
            cy.get(selectors.tooltip.bodyContent.uidFieldValue).should(
                'not.have.class',
                'text-alert-600'
            );
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('e7519642-959a-534b-8296-59de4560d4ab').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the process in baseline activity event details', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(
                selectors.eventTimeline.timeline.mainView.event.processInBaselineActivity
            ).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', '/bin/bash');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Process Activity');
            cy.get(selectors.tooltip.body).should('contain', 'Arguments: None');
            cy.get(selectors.tooltip.body).should('contain', 'UID: 0');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById('fafd4c56-a4e0-5fd9-aed2-c77b462ca637').then(
                (formattedEventTime) => {
                    cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
                }
            );
        });

        it('shows the container restart event details', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', 'nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Container Restart');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'abd2f41e72e825a76c2ab8898e538aa046872dd95a77a6c7d715881174f9e013'
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the container termination event details', () => {
            openMockedPodEventTimelineView();

            // trigger the tooltip
            cy.get(selectors.eventTimeline.timeline.mainView.event.termination).trigger(
                'mouseenter'
            );

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', 'nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Container Termination');
            cy.get(selectors.tooltip.body).should('contain', 'Reason: OOMKilled');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                '016963e1050fec95a53862373a6b5f0bff2a003cb9796ecfda492a9f7ce3214d'
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });
    });

    describe('Pagination', () => {
        it('should be able to page between sets of pods when there are 10+', () => {
            setRoutes();
            // mocking data to thoroughly test the pagination
            cy.intercept('POST', api.graphql(api.risks.graphqlOps.getPodEventTimeline), {
                fixture: 'risks/eventTimeline/podEventTimelineWithManyContainers.json',
            }).as('getPodEventTimeline');

            openEventTimeline();

            cy.wait('@getDeploymentEventTimeline');

            // click the button and drill down to see containers
            cy.get(selectors.eventTimeline.timeline.namesList.drillDownButtonInFirstRow).click();

            cy.wait('@getPodEventTimeline');

            // we should see the first 10 pods out of a total of 15
            cy.get(selectors.eventTimeline.timeline.namesList.listOfNames).should(
                'have.length',
                10
            );

            // go to the next page
            cy.get(selectors.eventTimeline.timeline.pagination.nextPage).click({ force: true });

            // we should see the last 5 pods out of the total of 15s
            cy.get(selectors.eventTimeline.timeline.namesList.listOfNames).should('have.length', 5);
        });
    });

    describe('Legend', () => {
        it('should show the timeline legend', () => {
            setRoutes();
            openEventTimeline();

            cy.wait('@getDeploymentEventTimeline');

            // click the button and drill down to see containers
            cy.get(selectors.eventTimeline.timeline.namesList.drillDownButtonInFirstRow).click();

            cy.wait('@getPodEventTimeline');

            // show the legend
            cy.get(selectors.eventTimeline.legend).click();

            // make sure the process activity icon and text shows up
            cy.get(`${selectors.tooltip.legendContents}:eq(0):contains("Process Activity")`);
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(0) ${selectors.tooltip.legendContent.event.processActivity}`
            );

            // make sure the policy violation icon and text shows up
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(1):contains("Process Activity with Violation")`
            );
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(1) ${selectors.tooltip.legendContent.event.policyViolation}`
            );

            // make sure the process in baseline activity icon and text shows up
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(2):contains("Baseline Process Activity")`
            );
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(2) ${selectors.tooltip.legendContent.event.processInBaselineActivity}`
            );

            // make sure the container restart icon and text shows up
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(3):contains("Container Restart")`
            );
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(3) ${selectors.tooltip.legendContent.event.restart}`
            );

            // make sure the container termination icon and text shows up
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(4):contains("Container Termination")`
            );
            cy.get(
                `${selectors.tooltip.legendContents} [data-testid="timeline-legend-items"] div:eq(4) ${selectors.tooltip.legendContent.event.termination}`
            );
        });
    });
});
