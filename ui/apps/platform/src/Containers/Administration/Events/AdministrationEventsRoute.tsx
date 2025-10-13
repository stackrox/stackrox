import React from 'react';
import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import AdministrationEventPage from './AdministrationEventPage';
import AdministrationEventsPage from './AdministrationEventsPage';

function AdministrationEventsRoute(): ReactElement {
    const { id } = useParams();

    if (id) {
        return <AdministrationEventPage id={id} />;
    }

    return <AdministrationEventsPage />;
}

export default AdministrationEventsRoute;
