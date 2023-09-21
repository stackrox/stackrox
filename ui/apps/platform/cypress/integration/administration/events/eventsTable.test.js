import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    assertDescriptionListGroup,
    eventAlias,
    eventsAlias,
    eventsCountAlias,
    visitAdministrationEventFromTableRow,
    visitAdministrationEvents,
} from './AdministrationEvents.helpers';

const event = {
    id: 'fd8dc19a-ab81-5f91-90b7-a6078f69e73f',
    type: 'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE',
    level: 'ADMINISTRATION_EVENT_LEVEL_ERROR',
    message:
        'error enriching image "quay.io/rhacs-eng/qa:nginx-unprivileged-1.15.12": image enrichment error: error getting metadata for image: quay.io/rhacs-eng/qa:nginx-unprivileged-1.15.12 error: getting metadata from registry: "Public Quay.io": failed to get the manifest digest: Head "https://quay.io/v2/rhacs-eng/qa/manifests/nginx-unprivileged-1.15.12": http: non-successful response (status=401 body="")',
    hint: 'An issue occurred scanning the image. Please ensure that:\n- Scanner can access the registry.\n- Correct credentials are configured for the particular registry / repository.\n- The scanned manifest exists within the registry / repository.',
    domain: 'Image Scanning',
    resourceType: 'Image',
    resourceId: 'quay.io/rhacs-eng/qa:nginx-unprivileged-1.15.12',
    numOccurrences: '1',
    lastOccurredAt: '2023-09-15T18:11:34.269927Z',
    createdAt: '2023-09-15T18:11:34.269927Z',
};

const staticResponseMapForEvent = {
    [eventAlias]: {
        body: {
            event,
        },
    },
};

const events = [event];

const staticResponseMapForEvents = {
    [eventsAlias]: {
        body: {
            events,
        },
    },
    [eventsCountAlias]: {
        body: {
            count: String(events.length),
        },
    },
};

describe('Administration Events table', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_ADMINISTRATION_EVENTS')) {
            this.skip();
        }
    });

    it('displays table head cells', () => {
        visitAdministrationEvents(staticResponseMapForEvents);

        cy.get('th:contains("Level")');
        cy.get('th:contains("Domain")');
        cy.get('th:contains("Resource type")');
        cy.get('th:contains("Event last occurred at")');
        cy.get('th:contains("Occurrences")');
    });

    it('has link to event page', () => {
        visitAdministrationEvents(staticResponseMapForEvents);
        visitAdministrationEventFromTableRow(0, staticResponseMapForEvent);

        cy.get('h1:contains("Error - Image Scanning")');
        assertDescriptionListGroup('Resource type', event.resourceType);
        assertDescriptionListGroup('Resource Id', event.resourceId);
        assertDescriptionListGroup('Event type', 'Log');
        assertDescriptionListGroup('Created at', event.createdAt);
        assertDescriptionListGroup('Last occurred at', event.lastOccurredAt);
        assertDescriptionListGroup('Occurrences', event.numOccurrences);
    });
});
