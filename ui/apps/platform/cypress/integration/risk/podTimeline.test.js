import withAuth from '../../helpers/basicAuth';

import {
    clickFirstDrillDownButtonInEventTimeline,
    clickNextPageInEventTimelineWithoutRequest,
    clickTab,
    filterEventsByType,
    getFormattedEventTimeById,
    viewGraph,
    viewRiskDeploymentByName,
    visitRiskDeployments,
} from './Risk.helpers';
import { selectors } from './Risk.selectors';

function openEventTimeline() {
    visitRiskDeployments();
    viewRiskDeploymentByName('collector');
    clickTab('Process Discovery');
    viewGraph();
}

const fixtureForDeploymentEventTimeline = 'risks/eventTimeline/deploymentEventTimeline.json';

const fixtureForPodEventTimeline = 'risks/eventTimeline/podEventTimeline.json';

describe('Risk Event Timeline for Pod', () => {
    withAuth();

    describe('Clustering Events', () => {
        it('should show the clustered event markers', () => {
            openEventTimeline();

            // mocking data to thoroughly test the clustering
            // click the button and drill down to see containers
            clickFirstDrillDownButtonInEventTimeline(
                'risks/eventTimeline/clusteredPodEventTimeline.json'
            );

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
            openEventTimeline();

            // mocking data to thoroughly test the filtering
            // click the button and drill down to see containers
            clickFirstDrillDownButtonInEventTimeline(
                'risks/eventTimeline/clusteredPodEventTimeline.json'
            );

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
        it('should filter policy violation events', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // the policy violation event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation);

            // filter by something else
            filterEventsByType('Process Activities');

            // the policy violation event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation).should(
                'not.exist'
            );
        });

        it('should filter process activity events and process in baseline activity events', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // the process activity + process in baseline activity event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.processActivity);
            cy.get(selectors.eventTimeline.timeline.mainView.event.processInBaselineActivity);

            // filter by something else
            filterEventsByType('Policy Violations');

            // the process activity + process in baseline activity event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.processActivity).should(
                'not.exist'
            );
            cy.get(
                selectors.eventTimeline.timeline.mainView.event.processInBaselineActivity
            ).should('not.exist');
        });

        it('should filter container restart events', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // the container restart event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart);

            // filter by something else
            filterEventsByType('Policy Violations');

            // thecontainer restart event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart).should('not.exist');
        });

        it('should filter container termination events', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // the container termination event should be visible
            cy.get(selectors.eventTimeline.select.value).should('contain', 'Show All');
            cy.get(selectors.eventTimeline.timeline.mainView.event.termination);

            // filter by something else
            filterEventsByType('Policy Violations');

            // the container termination event should not be visible
            cy.get(selectors.eventTimeline.timeline.mainView.event.containerTermination).should(
                'not.exist'
            );
        });
    });

    describe('Event Details', () => {
        it('shows the policy violation event details', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // trigger the tooltip
            cy.get(selectors.eventTimeline.timeline.mainView.event.policyViolation).trigger(
                'mouseenter'
            );

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', 'Ubuntu Package Manager Execution');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Policy Violation');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'd7a275e1-1bba-47e7-92a1-42340c759883',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the process activity event details for a process with no parent', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
            getFormattedEventTimeById(
                'e7519642-958a-534b-8297-59de4560d4ab',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the process activity event details for a process with a parent and unknown parent uid', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
            cy.get(selectors.tooltip.getUidFieldIconSelector('danger')).should('exist');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'e7519642-958a-534b-8246-59de4560d4ab',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the process activity event details for a process with a uid change', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
            cy.get(selectors.tooltip.getUidFieldIconSelector('danger')).should('exist');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'e7519642-958a-534b-8296-59de5560d4ab',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the process activity event details for a process with no uid change', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
            cy.get(selectors.tooltip.getUidFieldIconSelector('danger')).should('not.exist');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'e7519642-959a-534b-8296-59de4560d4ab',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the process in baseline activity event details', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
            getFormattedEventTimeById(
                'fafd4c56-a4e0-5fd9-aed2-c77b462ca637',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the container restart event details', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

            // trigger the tooltip
            cy.get(selectors.eventTimeline.timeline.mainView.event.restart).trigger('mouseenter');

            // the header should include the event name
            cy.get(selectors.tooltip.title).should('contain', 'nginx');
            // the body should include the following
            cy.get(selectors.tooltip.body).should('contain', 'Type: Container Restart');
            // since the displayed time depends on the time zone, we don't want to check against a  hardcoded value
            getFormattedEventTimeById(
                'abd2f41e72e825a76c2ab8898e538aa046872dd95a77a6c7d715881174f9e013',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });

        it('shows the container termination event details', () => {
            openEventTimeline();
            clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline);

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
                '016963e1050fec95a53862373a6b5f0bff2a003cb9796ecfda492a9f7ce3214d',
                fixtureForDeploymentEventTimeline
            ).then((formattedEventTime) => {
                cy.get(selectors.tooltip.body).should('contain', formattedEventTime);
            });
        });
    });

    describe('Pagination', () => {
        it('should be able to page between sets of pods when there are 10+', () => {
            openEventTimeline();

            // mocking data to thoroughly test the pagination
            // click the button and drill down to see containers
            clickFirstDrillDownButtonInEventTimeline(
                'risks/eventTimeline/podEventTimelineWithManyContainers.json'
            );

            // we should see the first 10 pods out of a total of 15
            cy.get(selectors.eventTimeline.timeline.namesList.listOfNames).should(
                'have.length',
                10
            );

            // go to the next page
            clickNextPageInEventTimelineWithoutRequest();

            // we should see the last 5 pods out of the total of 15s
            cy.get(selectors.eventTimeline.timeline.namesList.listOfNames).should('have.length', 5);
        });
    });

    describe('Legend', () => {
        it('should show the timeline legend', () => {
            openEventTimeline();

            // click the button and drill down to see containers
            clickFirstDrillDownButtonInEventTimeline();

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
